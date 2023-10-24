// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package e2e

import (
	"fmt"
	"testing"

	"github.com/DataDog/datadog-agent/test/new-e2e/pkg/utils/e2e"
	"github.com/DataDog/datadog-agent/test/new-e2e/pkg/utils/e2e/params"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/vm/ec2os"
	"github.com/DataDog/test-infra-definitions/scenarios/aws/vm/ec2params"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

type installFipsTestSuite struct {
	linuxInstallerTestSuite
}

func TestLinuxInstallFipsSuite(t *testing.T) {
	if flavor != agentFlavorDatadogAgent {
		t.Skip("fips test supports only datadog-agent flavor")
	}
	scriptType := "production"
	if scriptURL != defaultScriptURL {
		scriptType = "custom"
	}
	t.Run(fmt.Sprintf("We will install with fips %s with %s install_script on Ubuntu 22.04", flavor, scriptType), func(t *testing.T) {
		testSuite := &installFipsTestSuite{}
		e2e.Run(t,
			testSuite,
			e2e.EC2VMStackDef(ec2params.WithOS(ec2os.UbuntuOS)),
			params.WithStackName(fmt.Sprintf("install-fips-%s-ubuntu22", flavor)),
		)
	})
}

func (s *installFipsTestSuite) TestInstallFips() {
	t := s.T()
	vm := s.Env().VM
	t.Log("Install latest Agent 7 RC")
	cmd := fmt.Sprintf("DD_FIPS_MODE=true DD_URL=\"fake.url.com\" DD_AGENT_FLAVOR=%s DD_AGENT_MAJOR_VERSION=7 DD_API_KEY=%s DD_SITE=\"darth.vador.com\" DD_REPO_URL=datad0g.com DD_AGENT_DIST_CHANNEL=beta bash -c \"$(curl -L %s/install_script_agent7.sh)\"",
		flavor,
		apiKey,
		scriptURL)
	output := vm.Execute(cmd)

	s.assertInstallFips(output)
	s.uninstall()
	s.assertUninstall()
	s.purgeFips()
	s.assertPurge()
}

func (s *installFipsTestSuite) assertInstallFips(installCommandOutput string) {
	t := s.T()
	vm := s.Env().VM

	s.assertInstallScript()

	t.Log("assert install output contains expected lines")
	assert.Contains(t, installCommandOutput, "Installing package(s): datadog-agent datadog-signing-keys datadog-fips-proxy", "Missing installer log line for installing package(s)")
	assert.Contains(t, installCommandOutput, "* Adding your API key to the Datadog Agent configuration: /etc/datadog-agent/datadog.yaml", "Missing installer log line for API key")
	assert.Contains(t, installCommandOutput, "* Setting Datadog Agent configuration to use FIPS proxy: /etc/datadog-agent/datadog.yaml", "Missing installer log line for FIPS proxy")

	t.Log("assert agent configuration contains expected properties")
	configContent := vm.Execute(fmt.Sprintf("sudo cat /etc/%s/%s", s.baseName, s.configFile))
	var config map[string]any
	err := yaml.Unmarshal([]byte(configContent), &config)
	require.NoError(t, err, fmt.Sprintf("unexpected error on yaml parse %v", err))
	assert.Contains(t, config, "fips")
	fipsConfig, ok := config["fips"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, true, fipsConfig["enabled"])
	assert.Equal(t, 9803, fipsConfig["port_range_start"])
	assert.Equal(t, false, fipsConfig["https"])
	assert.Equal(t, apiKey, config["api_key"], "not matching api key in config")
	assert.NotContains(t, config, "site", "site modified in config")
	assert.NotContains(t, config, "dd_url", "dd_url modified in config")
}

func (s *installFipsTestSuite) purgeFips() {
	t := s.T()
	vm := s.Env().VM
	// Remove installed binary
	if _, err := vm.ExecuteWithError("command -v apt"); err != nil {
		t.Skip("Purge supported only with apt")
	}
	t.Log("Purge")
	vm.Execute(fmt.Sprintf("sudo apt remove --purge -y %s datadog-fips-proxy", flavor))
}