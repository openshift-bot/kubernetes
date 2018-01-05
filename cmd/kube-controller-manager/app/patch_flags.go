package app

import (
	"fmt"
	"io/ioutil"

	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/validation/field"
	kyaml "k8s.io/apimachinery/pkg/util/yaml"
	apiserverflag "k8s.io/apiserver/pkg/util/flag"
	"k8s.io/kubernetes/cmd/kube-controller-manager/app/config"
	"k8s.io/kubernetes/cmd/kube-controller-manager/app/options"
)

func getOpenShiftConfig(configFile string) (map[string]interface{}, error) {
	configBytes, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, err
	}
	jsonBytes, err := kyaml.ToJSON(configBytes)
	if err != nil {
		return nil, err
	}
	config := map[string]interface{}{}
	if err := json.Unmarshal(jsonBytes, &config); err != nil {
		return nil, err
	}

	return config, nil
}

func applyOpenShiftConfigFlags(controllerManagerOptions *options.KubeControllerManagerOptions, controllerManager *config.Config, openshiftConfig map[string]interface{}) error {
	kubeMasterConfig, ok := openshiftConfig["kubernetesMasterConfig"]
	if !ok {
		return nil
	}

	castKubeMasterConfig := kubeMasterConfig.(map[string]interface{})
	controllerArgs, ok := castKubeMasterConfig["controllerArguments"]
	if !ok || controllerArgs == nil {
		return nil
	}

	args := map[string][]string{}
	for key, value := range controllerArgs.(map[string]interface{}) {
		for _, arrayValue := range value.([]interface{}) {
			args[key] = append(args[key], arrayValue.(string))
		}
	}
	if err := applyFlags(args, controllerManagerOptions.Flags(KnownControllers(), ControllersDisabledByDefault.List())); len(err) > 0 {
		return kerrors.NewAggregate(err)
	}
	return controllerManagerOptions.ApplyTo(controllerManager, "kube-controller-manager")
}

// applyFlags stores the provided arguments onto a flag set, reporting any errors
// encountered during the process.
func applyFlags(args map[string][]string, flags apiserverflag.NamedFlagSets) []error {
	var errs []error
	for key, value := range args {
		for _, fs := range flags.FlagSets {
			flag := fs.Lookup(key)
			if flag == nil {
				errs = append(errs, field.Invalid(field.NewPath("flag"), key, "is not a valid flag"))
				continue
			}
			for _, s := range value {
				if err := flag.Value.Set(s); err != nil {
					errs = append(errs, field.Invalid(field.NewPath(key), s, fmt.Sprintf("could not be set: %v", err)))
					break
				}
			}
		}
	}
	return errs
}
