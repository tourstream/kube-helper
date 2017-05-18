package app

import "testing"

func TestCmdStartUpAllWithWrongConf(t *testing.T) {
	helperTestCmdHasWrongConfigReturned(t, CmdStartUpAll, []string{"startup-all", "-c", "never.yml"})
}

func TestCmdStartUpAllWithErrorForClientSet(t *testing.T) {
	helperTestCmdlWithErrorForClientSet(t, CmdStartUpAll, []string{"startup-all", "-c", "never.yml"})
}

func TestCmdStartUpAllWithErrorForClientSet(t *testing.T) {
	helperTestCmdlWithErrorForClientSet(t, CmdStartUpAll, []string{"startup-all", "-c", "never.yml"})
}

