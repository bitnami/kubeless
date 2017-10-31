package runtime

import (
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"k8s.io/client-go/pkg/api/v1"
)

func check(runtime, fname string, values []string, t *testing.T) {
	fileName := GetFunctionFileName(fname, runtime)
	depName, _ := GetRuntimeDepName(runtime)
	if depName != values[0] {
		t.Fatalf("Retrieving the image returned a wrong dependencies file. Received " + depName + " while expecting " + values[0])
	}
	if fileName != values[1] {
		t.Fatalf("Retrieving the image returned a wrong file name. Received " + fileName + " while expecting " + values[1])
	}
}

func TestGetFunctionFileNames(t *testing.T) {
	expectedValues := []string{"requirements.txt", "test.py"}
	check("python2.7", "test", expectedValues, t)
	check("python3.4", "test", expectedValues, t)

	expectedValues = []string{"package.json", "test.js"}
	check("nodejs6", "test", expectedValues, t)
	check("nodejs8", "test", expectedValues, t)

	expectedValues = []string{"Gemfile", "test.rb"}
	check("ruby2.4", "test", expectedValues, t)

	expectedValues = []string{"requirements.xml", "test.cs"}
	check("dotnetcore2.0", "test", expectedValues, t)
}

func TestGetFunctionImage(t *testing.T) {
	// Throws an error if the runtime doesn't exist
	_, err := GetFunctionImage("unexistent", "HTTP")
	if err == nil {
		t.Fatalf("Retrieving data for 'unexistent' should return an error")
	}

	// Throws an error if the runtime version doesn't exist
	_, err = GetFunctionImage("nodejs3", "HTTP")
	expectedErrMsg := regexp.MustCompile("The given runtime and version 'nodejs3' does not have a valid image")
	if expectedErrMsg.FindString(err.Error()) == "" {
		t.Fatalf("Retrieving data for 'nodejs3' should return an error")
	}

	expectedImageName := "ruby-test-image"
	os.Setenv("RUBY_RUNTIME", expectedImageName)
	imageR, errR := GetFunctionImage("ruby2.4", "HTTP")
	if errR != nil {
		t.Fatalf("Retrieving the image returned err: %v", errR)
	}
	if imageR != expectedImageName {
		t.Fatalf("Expecting " + imageR + " to be set to " + expectedImageName)
	}
	os.Unsetenv("RUBY_RUNTIME")

	expectedImageName = "ruby-pubsub-test-image"
	os.Setenv("RUBY_PUBSUB_RUNTIME", "ruby-pubsub-test-image")
	imageR, errR = GetFunctionImage("ruby2.4", "PubSub")
	if errR != nil {
		t.Fatalf("Retrieving the image returned err: %v", errR)
	}
	if imageR != expectedImageName {
		t.Fatalf("Expecting " + imageR + " to be set to " + expectedImageName)
	}
	os.Unsetenv("RUBY_PUBSUB_RUNTIME")
}

func TestGetRuntimes(t *testing.T) {
	runtimes := strings.Join(GetRuntimes(), ", ")
	expectedRuntimes := "python2.7, python3.4, nodejs6, nodejs8, ruby2.4, dotnetcore2.0"
	if runtimes != expectedRuntimes {
		t.Errorf("Expected %s but got %s", expectedRuntimes, runtimes)
	}
}

func TestGetBuildContainer(t *testing.T) {
	// It should throw an error if there is not an image available
	_, err := GetBuildContainer("notExists", []v1.EnvVar{}, v1.VolumeMount{}, v1.VolumeMount{})
	if err == nil {
		t.Error("Expected to throw an error")
	}

	// It should return the proper build image for python
	env := []v1.EnvVar{}
	vol1 := v1.VolumeMount{Name: "v1", MountPath: "/v1"}
	vol2 := v1.VolumeMount{Name: "v2", MountPath: "/v2"}
	c, err := GetBuildContainer("python2.7", env, vol1, vol2)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	expectedContainer := v1.Container{
		Name:            "install",
		Image:           "tuna/python-pillow:2.7.11-alpine",
		Command:         []string{"sh", "-c"},
		Args:            []string{"pip install --prefix=/v1 -r /v2/requirements.txt"},
		VolumeMounts:    []v1.VolumeMount{vol1, vol2},
		ImagePullPolicy: v1.PullIfNotPresent,
		Env:             env,
	}
	if !reflect.DeepEqual(expectedContainer, c) {
		t.Errorf("Unexpected result")
	}

	// It should return the proper build image for nodejs
	nodeEnv := []v1.EnvVar{
		{Name: "NPM_REGISTRY", Value: "http://reg.com"},
		{Name: "NPM_SCOPE", Value: "myorg"},
	}
	c, err = GetBuildContainer("nodejs6", nodeEnv, vol1, vol2)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if c.Image != "node:6.10" {
		t.Errorf("Unexpected image %s", c.Image)
	}
	if c.Args[0] != "npm config set myorg:registry http://reg.com && cd /v2 && npm install --prefix=/v1" {
		t.Errorf("Unexpected command %s", c.Args[0])
	}

	// It should return the proper build image for ruby
	c, err = GetBuildContainer("ruby2.4", env, vol1, vol2)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if c.Image != "bitnami/ruby:2.4" {
		t.Errorf("Unexpected image %s", c.Image)
	}
	if c.Args[0] != "bundle install --gemfile=/v2/Gemfile --path=/v1" {
		t.Errorf("Unexpected command %s", c.Args[0])
	}

}
