// Copyright 2019 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package maistra

import (
	"strings"
	"testing"
	"time"

	"istio.io/istio/pkg/log"
	"maistra/util"
)

func cleanup10(namespace, kubeconfig string) {
	log.Infof("# Cleanup. Following error can be ignored...")
	util.KubeDelete(namespace, httpbinTimeoutYaml, kubeconfig)
	util.KubeDelete(namespace, egressGoogleYaml, kubeconfig)
	util.KubeDelete(namespace, egressHTTPBinYaml, kubeconfig)
	util.KubeDelete(namespace, sleepIPRangeYaml, kubeconfig)
	util.KubeDelete(namespace, sleepYaml, kubeconfig)
	log.Info("Waiting for rules to be cleaned up. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
}

func configEgress(namespace, kubeconfig string) error {
	if err := util.KubeApply(namespace, egressHTTPBinYaml, kubeconfig); err != nil {
		return err
	}
	if err := util.KubeApply(namespace, egressGoogleYaml, kubeconfig); err != nil {
		return err
	}
	log.Info("Waiting for rules to propagate. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
	return nil
}

func blockExternal(namespace, kubeconfig string) error {
	if err := util.KubeApply(namespace, httpbinTimeoutYaml, kubeconfig); err != nil {
		return err
	}
	log.Info("Waiting for rules to propagate. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
	return nil
}

func deploySleepIPRange(namespace, kubeconfig string) error {
	log.Info("Deploy Sleep with IP Range")
	if err := util.KubeApply(namespace, sleepIPRangeYaml, kubeconfig); err != nil {
		return err
	}
	log.Info("Waiting for rules to propagate. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
	if err := util.CheckPodRunning(namespace, "app=sleep", kubeconfig); err != nil {
		return err
	}
	log.Info("Waiting for rules to propagate. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
	return nil
}

func Test10(t *testing.T) {
	defer cleanup10(testNamespace, kubeconfigFile)
	defer func() {
		// recover from panic if one occurred. This allows cleanup to be executed after panic.
		if err := recover(); err != nil {
			t.Errorf("Test panic: %v", err)
		}
	}()

	log.Infof("# TC_10 Control Egress Traffic")
	util.Inspect(deploySleep(testNamespace, kubeconfigFile), "failed to deploy sleep", "", t)
	util.Inspect(configEgress(testNamespace, kubeconfigFile), "failed to apply rules", "", t)
	pod, err := util.GetPodName(testNamespace, "app=sleep", kubeconfigFile)
	util.Inspect(err, "failed to get sleep pod", "", t)

	t.Run("external_httpbin", func(t *testing.T) {
		defer func() {
			// recover from panic if one occurred. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				log.Infof("Test panic: %v", err)
			}
		}()

		log.Info("# Make requests to external httpbin service")
		command := "curl http://httpbin.org/headers"
		msg, err := util.PodExec(testNamespace, pod, "sleep", command, false, kubeconfigFile)
		util.Inspect(err, "failed to get response", "", t)
		if strings.Contains(msg, "X-Envoy-Decorator-Operation") {
			log.Infof("Success. Get response header: %s", msg)

		} else {
			t.Errorf("Error response header: %s", msg)
		}

		log.Info("Waiting for rules to propagate. Sleep 10 seconds...")
		time.Sleep(time.Duration(10) * time.Second)
		/*
			logMsg := util.GetPodLogs(testNamespace, pod, "istio-proxy", true, false, kubeconfigFile)

			if strings.Contains(logMsg, "httpbin.org") {
				log.Infof("Get correct sidecar proxy log for httpbin.org")
			} else {
				t.Errorf("Error wrong sidecar proxy log for httpbin.org: %s", logMsg)
				log.Errorf("sidecar proxy log for httpbin.org: %s", logMsg)
			}

			logMsg = util.GetPodLogsForLabel("istio-system", "istio-mixer-type=telemetry", "mixer", false, false, kubeconfigFile)
			if strings.Contains(logMsg, "\"destinationServiceHost\":\"httpbin.org\"") {
				log.Infof("Get correct mixer log for httpbin.org")
			} else {
				t.Errorf("Error wrong mixer log for httpbin.org: %s", logMsg)
				log.Errorf("mixer log for httpbin.org: %s", logMsg)
			}
		*/
	})

	t.Run("external_google", func(t *testing.T) {
		defer func() {
			// recover from panic if one occurred. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				t.Errorf("Test panic: %v", err)
			}
		}()

		log.Info("# Make requets to external google service")
		command := "curl -s https://www.google.com | grep -o \"<title>.*</title>\""
		msg, err := util.PodExec(testNamespace, pod, "sleep", command, false, kubeconfigFile)
		util.Inspect(err, "failed to get response", "", t)
		if strings.Contains(msg, "<title>Google</title>") {
			log.Infof("Success. Get response: %s", msg)
		} else {
			t.Errorf("Error response: %s", msg)
		}

		/*
			logMsg := util.GetPodLogs(testNamespace, pod, "istio-proxy", true, false, kubeconfigFile)

			if strings.Contains(logMsg, "www.google.com") {
				log.Infof("Get correct sidecar proxy log for www.google.com")
			} else {

				log.Infof("sidecar proxy log for www.google.com: %s", logMsg)
			}

			log.Info("Waiting for rules to propagate. Sleep 10 seconds...")
			time.Sleep(time.Duration(10) * time.Second)
			logMsg = util.GetPodLogsForLabel("istio-system", "istio-mixer-type=telemetry", "mixer", true, false, kubeconfigFile)
			if strings.Contains(logMsg, "\"requestedServerName\":\"www.google.com\"") {
				log.Infof("Get correct mixer log for www.google.com")
			} else {
				t.Errorf("Error wrong mixer log for www.google.com: %s", logMsg)
				log.Errorf("mixer log for www.google.com: %s", logMsg)
			}
		*/
	})

	t.Run("block_external", func(t *testing.T) {
		defer func() {
			// recover from panic if one occurred. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				t.Errorf("Test panic: %v", err)
			}
		}()

		log.Info("Set route rules on external services")
		util.Inspect(blockExternal(testNamespace, kubeconfigFile), "failed to apply rules", "", t)
		command := "sh -c \"curl -o /dev/null -s -w '%{http_code}' http://httpbin.org/delay/5\""
		msg, err := util.PodExec(testNamespace, pod, "sleep", command, false, kubeconfigFile)
		util.Inspect(err, "failed to get response", "", t)
		if msg == "504" {
			log.Infof("Get expected response failure: %s", msg)
		} else {
			t.Errorf("Error response code: %s", msg)
		}
	})

	t.Run("bypass_ip_range", func(t *testing.T) {
		defer func() {
			// recover from panic if one occurred. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				t.Errorf("Test panic: %v", err)
			}
		}()

		cleanup10(testNamespace, kubeconfigFile)

		log.Info("# Redeploy sleep app with IP range exclusive and calling external services directly")
		util.Inspect(deploySleepIPRange(testNamespace, kubeconfigFile), "failed to deploy sleep with IP range", "", t)
		util.Inspect(configEgress(testNamespace, kubeconfigFile), "failed to apply rules", "", t)
		pod, err := util.GetPodName(testNamespace, "app=sleep", kubeconfigFile)
		util.Inspect(err, "failed to get sleep pod", "", t)

		t.Run("external_httpbin", func(t *testing.T) {
			log.Info("# Make requests to external httpbin service")
			log.Info("Waiting for rules to propagate. Sleep 50 seconds...")
			time.Sleep(time.Duration(50) * time.Second)

			command := "curl http://httpbin.org/headers"
			msg, err := util.PodExec(testNamespace, pod, "sleep", command, false, kubeconfigFile)
			util.Inspect(err, "failed to get response", "", t)
			if strings.Contains(msg, "X-Envoy-Decorator-Operation") {
				log.Errorf("Unexpected response header: %s", msg)
			} else {
				log.Infof("Success. Get response header without Istio sidecar: %s", msg)
			}
		})

		t.Run("external_google", func(t *testing.T) {
			defer func() {
				// recover from panic if one occurred. This allows cleanup to be executed after panic.
				if err := recover(); err != nil {
					t.Errorf("Test panic: %v", err)
				}
			}()

			log.Info("# Make requets to external google service")
			command := "curl -s https://www.google.com | grep -o \"<title>.*</title>\""
			msg, err := util.PodExec(testNamespace, pod, "sleep", command, false, kubeconfigFile)
			util.Inspect(err, "failed to get response", "", t)
			if strings.Contains(msg, "<title>Google</title>") {
				log.Infof("Success. Get response: %s", msg)
			} else {
				t.Errorf("Error response: %s", msg)
			}
		})
	})

}
