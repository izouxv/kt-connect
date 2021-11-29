package command

import (
	"errors"
	"flag"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"testing"

	"github.com/alibaba/kt-connect/pkg/kt"

	"github.com/alibaba/kt-connect/pkg/kt/options"
	"github.com/golang/mock/gomock"
	"github.com/urfave/cli"
)

func Test_meshCommand(t *testing.T) {

	ctl := gomock.NewController(t)
	fakeKtCli := kt.NewMockCliInterface(ctl)
	mockAction := NewMockActionInterface(ctl)

	mockAction.EXPECT().Mesh(gomock.Eq("service"), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	cases := []struct {
		testArgs               []string
		skipFlagParsing        bool
		useShortOptionHandling bool
		expectedErr            error
	}{
		{testArgs: []string{"mesh", "service", "--expose", "8080"}, skipFlagParsing: false, useShortOptionHandling: false, expectedErr: nil},
		{testArgs: []string{"mesh", "service"}, skipFlagParsing: false, useShortOptionHandling: false, expectedErr: errors.New("--expose is required")},
		{testArgs: []string{"mesh"}, skipFlagParsing: false, useShortOptionHandling: false, expectedErr: errors.New("name of deployment to mesh is required")},
	}

	for _, c := range cases {

		app := &cli.App{Writer: ioutil.Discard}
		set := flag.NewFlagSet("test", 0)
		_ = set.Parse(c.testArgs)

		context := cli.NewContext(app, set, nil)

		opts := options.NewDaemonOptions("test")
		opts.Debug = true
		command := NewMeshCommand(fakeKtCli, opts, mockAction)
		err := command.Run(context)

		if c.expectedErr != nil {
			require.Equal(t, err.Error(), c.expectedErr.Error(), "expected %v but is %v", c.expectedErr, err)
		} else {
			require.Equal(t, err, c.expectedErr, "expected %v but is %v", c.expectedErr, err)
		}
	}
}

func Test_toPortMapParameter(t *testing.T) {
	require.Equal(t, toPortMapParameter(map[int]int{ }), "", "port map parameter incorrect")
	require.Equal(t, toPortMapParameter(map[int]int{ 80:8080 }), "80:8080", "port map parameter incorrect")
	require.Equal(t, toPortMapParameter(map[int]int{ 80:8080, 70:7000 }), "80:8080,70:7000","port map parameter incorrect")
}

func Test_isValidKey(t *testing.T) {
	validCases := []string{"kt", "k123", "kt-version_1123"}
	for _, c := range validCases {
		require.True(t, isValidKey(c), "\"%s\" should be valid", c)
	}
	invalidCases := []string{"-version_1123", "123", "", "kt.ver"}
	for _, c := range invalidCases {
		require.True(t, !isValidKey(c), "\"%s\" should be invalid", c)
	}
}

func Test_getVersion(t *testing.T) {
	var k, v string
	k, v = getVersion("")
	require.Equal(t, k, "kt-version")
	require.Equal(t, len(v), 5)
	k, v = getVersion("test")
	require.Equal(t, k, "kt-version")
	require.Equal(t, v, "test")
	k, v = getVersion("mark:")
	require.Equal(t, k, "mark")
	require.Equal(t, len(v), 5)
	k, v = getVersion("mark:test")
	require.Equal(t, k, "mark")
	require.Equal(t, v, "test")
}
