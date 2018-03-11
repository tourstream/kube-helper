package testing

import (
	"errors"

	"gopkg.in/h2non/gock.v1"
	"k8s.io/apimachinery/pkg/runtime"
	testingK8s "k8s.io/client-go/testing"
)

func ErrorReturnFunc(action testingK8s.Action) (handled bool, ret runtime.Object, err error) {

	return true, nil, errors.New("explode")
}

func NilReturnFunc(action testingK8s.Action) (handled bool, ret runtime.Object, err error) {

	return true, nil, nil
}

func GetObjectReturnFunc(obj runtime.Object) testingK8s.ReactionFunc {
	return func(action testingK8s.Action) (handled bool, ret runtime.Object, err error) {

		return true, obj, nil
	}
}

func CreateAuthCall() {
	gock.New("https://accounts.google.com").
		Post("/o/oauth2/token").
		Reply(200).
		JSON(map[string]string{"access_token": "bar"})
}
