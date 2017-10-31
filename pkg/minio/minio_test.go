package minio

import (
	"fmt"
	"path"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	batchv1 "k8s.io/client-go/pkg/apis/batch/v1"
	core "k8s.io/client-go/testing"
)

func TestUploadFunction(t *testing.T) {
	// Fake successful job
	uploadFakeJob := batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "kubeless",
			Name:      "upload-file",
		},
		Status: batchv1.JobStatus{
			Succeeded: 1,
		},
	}
	file := "/path/to/func.ext"
	checksum := "abcdefghijklm1234567890"
	cli := &fake.Clientset{}
	cli.Fake.AddReactor("get", "jobs", func(action core.Action) (bool, runtime.Object, error) {
		return true, &uploadFakeJob, nil
	})

	// It should return a valid URL
	url, err := UploadFunction(file, checksum, cli)
	if err != nil {
		t.Errorf("Unexpected error %s", err)
	}
	if url != fmt.Sprintf("http://minio.kubeless:9000/functions/%s.%s", path.Base(file), checksum) {
		t.Errorf("Unexpected url %s", url)
	}
}
