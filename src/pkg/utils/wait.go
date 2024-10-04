// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2021-Present The Zarf Authors

// Package utils provides generic helper functions.
package utils

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/zarf-dev/zarf/src/api/v1alpha1"
	"github.com/zarf-dev/zarf/src/pkg/utils/exec"

	"github.com/zarf-dev/zarf/src/pkg/message"
)

// isJSONPathWaitType checks if the condition is a JSONPath or condition.
func isJSONPathWaitType(condition string) bool {
	if len(condition) == 0 || condition[0] != '{' || !strings.Contains(condition, "=") || !strings.Contains(condition, "}") {
		return false
	}

	return true
}

// ExecuteWait executes the wait-for command.
func ExecuteWait(waitTimeout, waitNamespace, condition, kind, identifier string, timeout time.Duration) error {
	// Handle network endpoints.
	switch kind {
	case "http", "https", "tcp":
		return waitForNetworkEndpoint(kind, identifier, condition, timeout)
	}

	// Type of wait, condition or JSONPath
	var waitType string

	// Check if waitType is JSONPath or condition
	if isJSONPathWaitType(condition) {
		waitType = "jsonpath="
	} else {
		waitType = "condition="
	}

	// Get the Zarf command configuration.
	zarfCommand, err := GetFinalExecutableCommand()
	if err != nil {
		return fmt.Errorf("could not locate the current Zarf binary path: %w", err)
	}

	// If the identifier contains an equals sign, convert to a label selector.
	if strings.ContainsRune(identifier, '=') {
		identifier = fmt.Sprintf("-l %s", identifier)
	}

	// Set the timeout for the wait-for command.
	expired := time.After(timeout)

	// Set the custom message for optional namespace.
	namespaceFlag := ""
	if waitNamespace != "" {
		namespaceFlag = fmt.Sprintf("-n %s", waitNamespace)
	}

	// Get the OS shell to execute commands in
	shell, shellArgs := exec.GetOSShell(v1alpha1.Shell{Windows: "cmd"})

	for {
		// Delay the check for 1 second
		time.Sleep(time.Second)

		select {
		case <-expired:
			return errors.New("wait timed out")

		default:
			// Check if the resource exists.
			zarfKubectlGet := fmt.Sprintf("%s tools kubectl get %s %s %s", zarfCommand, namespaceFlag, kind, identifier)
			stdout, stderr, err := exec.Cmd(shell, append(shellArgs, zarfKubectlGet)...)
			if err != nil {
				message.Debug(stdout, stderr, err)
				continue
			}

			resourceNotFound := strings.Contains(stderr, "No resources found") && identifier == ""
			if resourceNotFound {
				message.Debug(stdout, stderr, err)
				continue
			}

			// If only checking for existence, exit here.
			switch condition {
			case "", "exist", "exists":
				return nil
			}

			// Wait for the resource to meet the given condition.
			zarfKubectlWait := fmt.Sprintf("%s tools kubectl wait %s %s %s --for %s%s --timeout=%s",
				zarfCommand, namespaceFlag, kind, identifier, waitType, condition, waitTimeout)

			// If there is an error, log it and try again.
			if stdout, stderr, err := exec.Cmd(shell, append(shellArgs, zarfKubectlWait)...); err != nil {
				message.Debug(stdout, stderr, err)
				continue
			}

			// And just like that, success!
			return nil
		}
	}
}

// waitForNetworkEndpoint waits for a network endpoint to respond.
func waitForNetworkEndpoint(resource, name, condition string, timeout time.Duration) error {
	// Set the timeout for the wait-for command.
	expired := time.After(timeout)

	condition = strings.ToLower(condition)
	if condition == "" {
		condition = "success"
	}
	message.Debug(fmt.Sprintf("Waiting for network endpoint %s://%s to respond %s.", resource, name, condition))

	delay := 100 * time.Millisecond

	for {
		// Delay the check for 100ms the first time and then 1 second after that.
		time.Sleep(delay)
		delay = time.Second

		select {
		case <-expired:
			return errors.New("wait timed out")
		default:
			switch resource {
			case "http", "https":
				// Handle HTTP and HTTPS endpoints.
				url := fmt.Sprintf("%s://%s", resource, name)

				// Default to checking for a 2xx response.
				if condition == "success" {
					// Try to get the URL and check the status code.
					resp, err := http.Get(url)

					// If the status code is not in the 2xx range, try again.
					if err != nil || resp.StatusCode < 200 || resp.StatusCode > 299 {
						message.Debug(err)
						continue
					}

					// Success, break out of the switch statement.
					break
				}

				// Convert the condition to an int and check if it's a valid HTTP status code.
				code, err := strconv.Atoi(condition)
				if err != nil {
					return fmt.Errorf("http status code %s is not an integer: %w", condition, err)
				}
				if http.StatusText(code) == "" {
					return errors.New("http status code %s is unknown")
				}

				// Try to get the URL and check the status code.
				resp, err := http.Get(url)
				if err != nil || resp.StatusCode != code {
					message.Debug(err)
					continue
				}
			default:
				// Fallback to any generic protocol using net.Dial
				conn, err := net.Dial(resource, name)
				if err != nil {
					message.Debug(err)
					continue
				}
				err = conn.Close()
				if err != nil {
					message.Debug(err)
					continue
				}
			}

			// Yay, we made it!
			return nil
		}
	}
}
