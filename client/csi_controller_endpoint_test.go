package client

import (
	"errors"
	"testing"

	"github.com/hashicorp/nomad/client/dynamicplugins"
	"github.com/hashicorp/nomad/client/structs"
	nstructs "github.com/hashicorp/nomad/nomad/structs"
	"github.com/hashicorp/nomad/plugins/csi"
	"github.com/hashicorp/nomad/plugins/csi/fake"
	"github.com/stretchr/testify/require"
)

var fakePlugin = &dynamicplugins.PluginInfo{
	Name:           "test-plugin",
	Type:           "csi-controller",
	ConnectionInfo: &dynamicplugins.PluginConnectionInfo{},
}

func TestCSIController_AttachVolume(t *testing.T) {
	t.Parallel()

	cases := []struct {
		Name             string
		ClientSetupFunc  func(*fake.Client)
		Request          *structs.ClientCSIControllerAttachVolumeRequest
		ExpectedErr      error
		ExpectedResponse *structs.ClientCSIControllerAttachVolumeResponse
	}{
		{
			Name: "returns plugin not found errors",
			Request: &structs.ClientCSIControllerAttachVolumeRequest{
				PluginName: "some-garbage",
			},
			ExpectedErr: errors.New("plugin some-garbage for type csi-controller not found"),
		},
		{
			Name: "validates volumeid is not empty",
			Request: &structs.ClientCSIControllerAttachVolumeRequest{
				PluginName: fakePlugin.Name,
			},
			ExpectedErr: errors.New("VolumeID is required"),
		},
		{
			Name: "validates nodeid is not empty",
			Request: &structs.ClientCSIControllerAttachVolumeRequest{
				PluginName: fakePlugin.Name,
				VolumeID:   "1234-4321-1234-4321",
			},
			ExpectedErr: errors.New("NodeID is required"),
		},
		{
			Name: "validates AccessMode",
			Request: &structs.ClientCSIControllerAttachVolumeRequest{
				PluginName: fakePlugin.Name,
				VolumeID:   "1234-4321-1234-4321",
				NodeID:     "abcde",
				AccessMode: nstructs.CSIVolumeAccessMode("foo"),
			},
			ExpectedErr: errors.New("Unknown access mode: foo"),
		},
		{
			Name: "validates attachmentmode is not empty",
			Request: &structs.ClientCSIControllerAttachVolumeRequest{
				PluginName:     fakePlugin.Name,
				VolumeID:       "1234-4321-1234-4321",
				NodeID:         "abcde",
				AccessMode:     nstructs.CSIVolumeAccessModeMultiNodeReader,
				AttachmentMode: nstructs.CSIVolumeAttachmentMode("bar"),
			},
			ExpectedErr: errors.New("Unknown attachment mode: bar"),
		},
		{
			Name: "returns transitive errors",
			ClientSetupFunc: func(fc *fake.Client) {
				fc.NextControllerPublishVolumeErr = errors.New("hello")
			},
			Request: &structs.ClientCSIControllerAttachVolumeRequest{
				PluginName:     fakePlugin.Name,
				VolumeID:       "1234-4321-1234-4321",
				NodeID:         "abcde",
				AccessMode:     nstructs.CSIVolumeAccessModeSingleNodeWriter,
				AttachmentMode: nstructs.CSIVolumeAttachmentModeFilesystem,
			},
			ExpectedErr: errors.New("hello"),
		},
		{
			Name: "handles nil PublishContext",
			ClientSetupFunc: func(fc *fake.Client) {
				fc.NextControllerPublishVolumeResponse = &csi.ControllerPublishVolumeResponse{}
			},
			Request: &structs.ClientCSIControllerAttachVolumeRequest{
				PluginName:     fakePlugin.Name,
				VolumeID:       "1234-4321-1234-4321",
				NodeID:         "abcde",
				AccessMode:     nstructs.CSIVolumeAccessModeSingleNodeWriter,
				AttachmentMode: nstructs.CSIVolumeAttachmentModeFilesystem,
			},
			ExpectedResponse: &structs.ClientCSIControllerAttachVolumeResponse{},
		},
		{
			Name: "handles non-nil PublishContext",
			ClientSetupFunc: func(fc *fake.Client) {
				fc.NextControllerPublishVolumeResponse = &csi.ControllerPublishVolumeResponse{
					PublishContext: map[string]string{"foo": "bar"},
				}
			},
			Request: &structs.ClientCSIControllerAttachVolumeRequest{
				PluginName:     fakePlugin.Name,
				VolumeID:       "1234-4321-1234-4321",
				NodeID:         "abcde",
				AccessMode:     nstructs.CSIVolumeAccessModeSingleNodeWriter,
				AttachmentMode: nstructs.CSIVolumeAttachmentModeFilesystem,
			},
			ExpectedResponse: &structs.ClientCSIControllerAttachVolumeResponse{
				PublishContext: map[string]string{"foo": "bar"},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			require := require.New(t)
			client, cleanup := TestClient(t, nil)
			defer cleanup()

			fakeClient := &fake.Client{}
			if tc.ClientSetupFunc != nil {
				tc.ClientSetupFunc(fakeClient)
			}

			dispenserFunc := func(*dynamicplugins.PluginInfo) (interface{}, error) {
				return fakeClient, nil
			}
			client.dynamicRegistry.StubDispenserForType(dynamicplugins.PluginTypeCSIController, dispenserFunc)

			err := client.dynamicRegistry.RegisterPlugin(fakePlugin)
			require.Nil(err)

			var resp structs.ClientCSIControllerAttachVolumeResponse
			err = client.ClientRPC("CSIController.AttachVolume", tc.Request, &resp)
			require.Equal(tc.ExpectedErr, err)
			if tc.ExpectedResponse != nil {
				require.Equal(tc.ExpectedResponse, &resp)
			}
		})
	}
}

func TestClientCSI_CSIControllerValidateVolume(t *testing.T) {
	t.Parallel()

	cases := []struct {
		Name             string
		ClientSetupFunc  func(*fake.Client)
		Request          *structs.ClientCSIControllerValidateVolumeRequest
		ExpectedErr      error
		ExpectedResponse *structs.ClientCSIControllerValidateVolumeResponse
	}{
		{
			Name: "validates volumeid is not empty",
			Request: &structs.ClientCSIControllerValidateVolumeRequest{
				PluginID: fakePlugin.Name,
			},
			ExpectedErr: errors.New("VolumeID is required"),
		},
		{
			Name: "returns plugin not found errors",
			Request: &structs.ClientCSIControllerValidateVolumeRequest{
				PluginID: "some-garbage",
				VolumeID: "foo",
			},
			ExpectedErr: errors.New("plugin some-garbage for type csi-controller not found"),
		},
		{
			Name: "validates attachmentmode",
			Request: &structs.ClientCSIControllerValidateVolumeRequest{
				PluginID:       fakePlugin.Name,
				VolumeID:       "1234-4321-1234-4321",
				AttachmentMode: nstructs.CSIVolumeAttachmentMode("bar"),
				AccessMode:     nstructs.CSIVolumeAccessModeMultiNodeReader,
			},
			ExpectedErr: errors.New("Unknown volume attachment mode: bar"),
		},
		{
			Name: "validates AccessMode",
			Request: &structs.ClientCSIControllerValidateVolumeRequest{
				PluginID:       fakePlugin.Name,
				VolumeID:       "1234-4321-1234-4321",
				AttachmentMode: nstructs.CSIVolumeAttachmentModeFilesystem,
				AccessMode:     nstructs.CSIVolumeAccessMode("foo"),
			},
			ExpectedErr: errors.New("Unknown volume access mode: foo"),
		},
		{
			Name: "returns transitive errors",
			ClientSetupFunc: func(fc *fake.Client) {
				fc.NextControllerValidateVolumeErr = errors.New("hello")
			},
			Request: &structs.ClientCSIControllerValidateVolumeRequest{
				PluginID:       fakePlugin.Name,
				VolumeID:       "1234-4321-1234-4321",
				AccessMode:     nstructs.CSIVolumeAccessModeSingleNodeWriter,
				AttachmentMode: nstructs.CSIVolumeAttachmentModeFilesystem,
			},
			ExpectedErr: errors.New("hello"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			require := require.New(t)
			client, cleanup := TestClient(t, nil)
			defer cleanup()

			fakeClient := &fake.Client{}
			if tc.ClientSetupFunc != nil {
				tc.ClientSetupFunc(fakeClient)
			}

			dispenserFunc := func(*dynamicplugins.PluginInfo) (interface{}, error) {
				return fakeClient, nil
			}
			client.dynamicRegistry.StubDispenserForType(dynamicplugins.PluginTypeCSIController, dispenserFunc)

			err := client.dynamicRegistry.RegisterPlugin(fakePlugin)
			require.Nil(err)

			var resp structs.ClientCSIControllerValidateVolumeResponse
			err = client.ClientRPC("ClientCSI.CSIControllerValidateVolume", tc.Request, &resp)
			require.Equal(tc.ExpectedErr, err)
			if tc.ExpectedResponse != nil {
				require.Equal(tc.ExpectedResponse, &resp)
			}
		})
	}
}