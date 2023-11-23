package property

// import (
// 	"testing"

// 	externalsecretsv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
// 	"github.com/google/go-cmp/cmp"
// 	// metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
// )

// func removeMatchingProperties_test_fn[T any](jsonpath string, in *ChangeSet[T], want *ChangeSet[T], wantErr bool, t *testing.T) {
// 	if err := in.removeMatchingProperties(jsonpath); (err != nil) != wantErr {
// 		t.Errorf("ChangeSet_removeMatchingProperties() error = %v, wantErr %v", err, wantErr)
// 		return
// 	}
// 	if diff := cmp.Diff(in.current, want.current); len(diff) > 0 {
// 		t.Errorf("ChangeSet_removeMatchingProperties() diff in set.current = %v", diff)
// 	}
// 	if diff := cmp.Diff(in.desired, want.desired); len(diff) > 0 {
// 		t.Errorf("ChangeSet_removeMatchingProperties() diff in set.desired = %v", diff)
// 	}
// }

// func Test_ChangeSet_removeMatchingProperties(t *testing.T) {

// 	tests1 := []struct {
// 		name      string
// 		changeset *ChangeSet[externalsecretsv1beta1.ExternalSecretSpec]
// 		jsonpath  string
// 		want      *ChangeSet[externalsecretsv1beta1.ExternalSecretSpec]
// 		wantErr   bool
// 	}{
// 		{
// 			name: "Ignoress the specified nested propery",
// 			changeset: &ChangeSet[externalsecretsv1beta1.ExternalSecretSpec]{
// 				path: "spec",
// 				current: &externalsecretsv1beta1.ExternalSecretSpec{
// 					Data: []externalsecretsv1beta1.ExternalSecretData{
// 						{
// 							SecretKey: "ENVVAR1",
// 							RemoteRef: externalsecretsv1beta1.ExternalSecretDataRemoteRef{
// 								Key:            "some-key1",
// 								MetadataPolicy: externalsecretsv1beta1.ExternalSecretMetadataPolicyNone,
// 								Property:       "some-property1",
// 							},
// 						},
// 						{
// 							SecretKey: "ENVVAR2",
// 							RemoteRef: externalsecretsv1beta1.ExternalSecretDataRemoteRef{
// 								Key:            "some-key2",
// 								MetadataPolicy: externalsecretsv1beta1.ExternalSecretMetadataPolicyNone,
// 								Property:       "some-property2",
// 							},
// 						},
// 					},
// 				},
// 				desired: &externalsecretsv1beta1.ExternalSecretSpec{
// 					Data: []externalsecretsv1beta1.ExternalSecretData{
// 						{
// 							SecretKey: "ENVVAR1",
// 							RemoteRef: externalsecretsv1beta1.ExternalSecretDataRemoteRef{
// 								Key:            "some-key1",
// 								MetadataPolicy: "",
// 								Property:       "some-other-property1",
// 							},
// 						},
// 					},
// 				},
// 			},
// 			jsonpath: "spec.data[*].remoteRef.metadataPolicy",
// 			want: &ChangeSet[externalsecretsv1beta1.ExternalSecretSpec]{
// 				path: "spec",
// 				current: &externalsecretsv1beta1.ExternalSecretSpec{
// 					Data: []externalsecretsv1beta1.ExternalSecretData{
// 						{
// 							SecretKey: "ENVVAR1",
// 							RemoteRef: externalsecretsv1beta1.ExternalSecretDataRemoteRef{
// 								Key:      "some-key1",
// 								Property: "some-property1",
// 							},
// 						},
// 						{
// 							SecretKey: "ENVVAR2",
// 							RemoteRef: externalsecretsv1beta1.ExternalSecretDataRemoteRef{
// 								Key:      "some-key2",
// 								Property: "some-property2",
// 							},
// 						},
// 					},
// 				},
// 				desired: &externalsecretsv1beta1.ExternalSecretSpec{
// 					Data: []externalsecretsv1beta1.ExternalSecretData{
// 						{
// 							SecretKey: "ENVVAR1",
// 							RemoteRef: externalsecretsv1beta1.ExternalSecretDataRemoteRef{
// 								Key:      "some-key1",
// 								Property: "some-other-property1",
// 							},
// 						},
// 					},
// 				},
// 			},
// 			wantErr: false,
// 		},
// 	}
// 	for _, tt := range tests1 {
// 		t.Run(tt.name, func(t *testing.T) {
// 			removeMatchingProperties_test_fn(tt.jsonpath, tt.changeset, tt.want, tt.wantErr, t)
// 		})
// 	}

// 	tests2 := []struct {
// 		name      string
// 		changeset *ChangeSet[map[string]string]
// 		jsonpath  string
// 		want      *ChangeSet[map[string]string]
// 		wantErr   bool
// 	}{
// 		{
// 			name: "",
// 			changeset: &ChangeSet[map[string]string]{
// 				path: "metadata.annotations",
// 				current: &map[string]string{
// 					"deployment.kubernetes.io/revision": "36",
// 				},
// 				desired: &map[string]string{
// 					"key": "value",
// 				},
// 			},
// 			jsonpath: `metadata.annotations['deployment.kubernetes.io/revision']`,
// 			want: &ChangeSet[map[string]string]{
// 				path:    "metadata.annotations",
// 				current: &map[string]string{},
// 				desired: &map[string]string{
// 					"key": "value",
// 				},
// 			},
// 			wantErr: false,
// 		},
// 	}
// 	for _, tt := range tests2 {
// 		t.Run(tt.name, func(t *testing.T) {
// 			removeMatchingProperties_test_fn(tt.jsonpath, tt.changeset, tt.want, tt.wantErr, t)
// 		})
// 	}
// }
