package main

import (
	"context"
	"fmt"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"log"
	"os"
	"strconv"
	"strings"
)

type JobManager struct {
	clientset *kubernetes.Clientset
	jobImage  string
	ttl       int32
	namespace string
}

func NewJobManager() *JobManager {
	ji := os.Getenv("JOB_IMAGE")
	if ji == "" {
		log.Fatalf("Unable to find JOB_IMAGE in env vars")
	}

	t := os.Getenv("JOB_TTL")
	if t == "" {
		log.Fatalf("Unable to find JOB_TTL in env vars")
	}
	ttl, err := strconv.ParseInt(t, 10, 32)
	if err != nil {
		log.Fatalf("Unable to parse TTL \"%s\" to int", t)
	}

	ns := os.Getenv("NAMESPACE")
	if ns == "" {
		log.Fatalf("Unable to find NAMESPACE in env vars")
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	jm := &JobManager{
		clientset: clientset,
		ttl:       int32(ttl),
		jobImage:  ji,
		namespace: ns,
	}
	return jm
}

func (jm *JobManager) CreateJob(crawlID string, url string) {
	jobs := jm.clientset.BatchV1().Jobs(jm.namespace)
	var backOffLimit int32 = 0
	cmd := fmt.Sprintf("/sitemapper -s %s --id %s", url, crawlID)

	jobSpec := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("crawl-job-%s", crawlID),
			Namespace: jm.namespace,
			Labels: map[string]string{
				"crawl-id": crawlID,
			},
		},
		Spec: batchv1.JobSpec{
			TTLSecondsAfterFinished: &jm.ttl,
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("crawl-pod-%s", crawlID),
					Namespace: jm.namespace,
					Labels: map[string]string{
						"crawl-id": crawlID,
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:    "sitemapper",
							Image:   jm.jobImage,
							Command: strings.Split(cmd, " "),
							EnvFrom: []v1.EnvFromSource{{
								ConfigMapRef: &v1.ConfigMapEnvSource{
									LocalObjectReference: v1.LocalObjectReference{
										Name: "sitemapper",
									},
								},
							}},
						},
					},
					RestartPolicy: v1.RestartPolicyNever,
				},
			},
			BackoffLimit: &backOffLimit,
		},
	}

	j, err := jobs.Create(context.TODO(), jobSpec, metav1.CreateOptions{})
	if err != nil {
		log.Printf("Failed to create job: %s\n", err)
		return
	}

	log.Printf("Created job %s successfully", j.Name)
}
