/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	apiV1 "a-cool-name.io/k8s/api/v1"
	backupV1 "a-cool-name.io/k8s/api/v1"
	batchv1 "k8s.io/api/batch/v1"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

// BackupReconciler reconciles a Backup object
type BackupReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=task.a-cool-name.io,resources=backups,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=task.a-cool-name.io,resources=backups/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=task.a-cool-name.io,resources=backups/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Backup object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.0/pkg/reconcile
func (r *BackupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	k8sLogger := log.FromContext(ctx)

	k8sLogger.Info("⚡️ Event received! ⚡️")
	k8sLogger.Info("Request: ", "req", req)

	backup := &backupV1.Backup{}
	err := r.Get(ctx, req.NamespacedName, backup)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if backup.Status.Phase == "" {
		backup.Status.Phase = backupV1.BackupPhasePending
		err = r.Status().Update(ctx, backup)
		if err != nil {
			k8sLogger.Error(err, " ⛔ Stopping, unable to update Backup status")
			return ctrl.Result{}, err
		}
	}

	switch backup.Status.Phase {
	case backupV1.BackupPhasePending:
		k8sLogger.Info("⏳ BackupPhasePending")

		// Create a new Job/Cronjob
		switch backup.Spec.Type {
		case apiV1.BackupTypeCronjob:
			err := createAndRunCronJob(ctx, req, *backup, r)
			if err != nil {
				k8sLogger.Error(err, " ⛔ Stopping,unable to create CronJob")
				return ctrl.Result{}, err
			}
		case apiV1.BackupTypeJob:
			err := createAndRunJob(ctx, req, *backup, r)
			if err != nil {
				k8sLogger.Error(err, " ⛔ Stopping,unable to create Job")
				return ctrl.Result{}, err
			}
		default:
			err := fmt.Errorf(" ⛔ Stopping, Backup type unknown")
			k8sLogger.Error(err, err.Error())
			return ctrl.Result{}, err
		}

		backup.Status.Phase = backupV1.BackupPhaseRunning
	case backupV1.BackupPhaseRunning:
		k8sLogger.Info(" 👌 Backup Phase Running")
		err := listenForChangesInJob(ctx, r, req, backup)
		if err != nil {
			k8sLogger.Error(err, " ⛔ Stopping, unable to listen for changes in Job")
			return ctrl.Result{}, err
		}

		backup.Status.Phase = backupV1.BackupPhaseSucceeded
	case backupV1.BackupPhaseSucceeded:
		k8sLogger.Info(" 💚 Backup Phase Succeeded")

	default:
		k8sLogger.Info(" ⛔ Backup Phase Unknown")
	}

	err = r.Status().Update(ctx, backup)
	if err != nil {
		k8sLogger.Error(err, "unable to update Backup status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func createAndRunJob(ctx context.Context, req ctrl.Request, backup backupV1.Backup, r *BackupReconciler) error {
	k8sLogger := log.FromContext(ctx)
	job := createJobDefinition(backup)
	var existingJob batchv1.Job
	err := r.Get(ctx, req.NamespacedName, &existingJob)
	if !errors.IsNotFound(err) {
		k8sLogger.Error(err, "unable to fetch Job")
		return err
	}
	err = r.Create(ctx, &job)
	if err != nil {
		k8sLogger.Error(err, " ⛔ Stopping,unable to create Job")
		return err
	}
	err = ctrl.SetControllerReference(&backup, &job, r.Scheme)
	if err != nil {
		k8sLogger.Error(err, " ⛔ Stopping,unable to set owner reference on Job")
		return err
	}
	return nil
}

func createAndRunCronJob(ctx context.Context, req ctrl.Request, backup backupV1.Backup, r *BackupReconciler) error {
	k8sLogger := log.FromContext(ctx)

	if len(backup.Spec.Schedule) == 0 {
		err := fmt.Errorf(" ⛔ Stopping, Schedule is required for CronJob")
		k8sLogger.Error(err, err.Error())
		return err
	}

	cronJob := createCronJobDefinition(backup)
	var existingCronJob batchv1.CronJob
	err := r.Get(ctx, req.NamespacedName, &existingCronJob)
	if !errors.IsNotFound(err) {
		k8sLogger.Error(err, "unable to fetch CronJob")
		return err
	}
	err = r.Create(ctx, &cronJob)
	if err != nil {
		k8sLogger.Error(err, " ⛔ Stopping,unable to create CronJob")
		return err
	}
	err = ctrl.SetControllerReference(&backup, &cronJob, r.Scheme)
	if err != nil {
		k8sLogger.Error(err, " ⛔ Stopping,unable to set owner reference on CronJob")
		return err
	}
	return nil
}

func listenForChangesInJob(ctx context.Context, r *BackupReconciler, req ctrl.Request, backup *backupV1.Backup) error {
	k8sLogger := log.FromContext(ctx)
	config, err := config.GetConfig()
	if err != nil {
		k8sLogger.Error(err, " ⛔ Stopping, unable to get kubernetes config")
		return err
	}
	k8sclient, err := kubernetes.NewForConfig(config)
	if err != nil {
		k8sLogger.Error(err, " ⛔ Stopping, unable to create kubernetes client")
		return err
	}

	watchJob, err := k8sclient.BatchV1().Jobs(backup.Namespace).Watch(ctx, metav1.SingleObject(metav1.ObjectMeta{
		Name:      backup.Name,
		Namespace: backup.Namespace,
	}))
	if err != nil {
		k8sLogger.Error(err, " ⛔ Stopping, unable to watch job changes")
		return err
	}

	for {
		jobChanges := <-watchJob.ResultChan()
		if jobChanges.Object == nil {
			return fmt.Errorf("could not watch job changes, job event: %s", jobChanges.Type)
		}

		activejob := jobChanges.Object.(*batchv1.Job)

		if activejob.Status.Succeeded == 1 {
			backup.Status.Phase = backupV1.BackupPhaseSucceeded
			backup.Status.Conditions = []backupV1.BackupCondition{
				{
					Status:         "True",
					Type:           "Ready",
					LastUpdateTime: metav1.Now(),
					Message:        activejob.Status.Conditions[0].Message,
					Reason:         activejob.Status.Conditions[0].Reason,
				},
			}
			err = r.Status().Update(ctx, backup)
			if err != nil {
				k8sLogger.Error(err, " ⛔ Stopping, unable to update Backup status")
				return err
			}
			return nil
		} else if activejob.Status.Failed == 1 {
			backup.Status.Phase = backupV1.BackupPhaseFailed
			backup.Status.Conditions = []backupV1.BackupCondition{
				{
					Status:         "False",
					Type:           "Failed",
					LastUpdateTime: metav1.Now(),
					Message:        "😨 Backup failed",
					Reason:         activejob.Status.Conditions[0].Reason,
				},
			}
			err = r.Status().Update(ctx, backup)
			if err != nil {
				k8sLogger.Error(err, " ⛔ Stopping, unable to update Backup status")
				return err
			}
			return nil
		}
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *BackupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&backupV1.Backup{}).
		Owns(&batchv1.Job{}).
		Owns(&batchv1.CronJob{}).
		Complete(r)
}

func createCronJobDefinition(backup backupV1.Backup) batchv1.CronJob {
	cronjob := batchv1.CronJob{}
	cronjob.Name = backup.Name
	cronjob.Namespace = backup.Namespace
	cronjob.Spec.Schedule = backup.Spec.Schedule
	cronjob.Spec.JobTemplate.Spec.Template.Spec.Containers = []coreV1.Container{
		{
			Name:  "backup",
			Image: "busybox",
			Args:  []string{"echo", " 😎 doing my backup cronjob"},
		},
	}
	cronjob.Spec.JobTemplate.Spec.Template.Spec.RestartPolicy = coreV1.RestartPolicyOnFailure

	return cronjob
}

func createJobDefinition(backup backupV1.Backup) batchv1.Job {
	job := batchv1.Job{}
	job.Name = backup.Name
	job.Namespace = backup.Namespace
	job.Spec.Template.Spec.Containers = []coreV1.Container{
		{
			Name:  "backup",
			Image: "busybox",
			Args:  []string{"echo", " 😍 doing my backup job"},
		},
	}
	job.Spec.Template.Spec.RestartPolicy = coreV1.RestartPolicyOnFailure

	return job
}
