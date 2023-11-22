package property

import (
	"testing"

	externalsecretsv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/google/go-cmp/cmp"
	// metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_ChangeSet_removeMatchingProperties(t *testing.T) {
	tests := []struct {
		name      string
		changeset *ChangeSet[externalsecretsv1beta1.ExternalSecretSpec]
		path      string
		want      *ChangeSet[externalsecretsv1beta1.ExternalSecretSpec]
		wantErr   bool
	}{
		{
			name: "Ignoress the specified nested propery",
			changeset: &ChangeSet[externalsecretsv1beta1.ExternalSecretSpec]{
				path: "spec",
				current: &externalsecretsv1beta1.ExternalSecretSpec{
					Data: []externalsecretsv1beta1.ExternalSecretData{
						{
							SecretKey: "ENVVAR1",
							RemoteRef: externalsecretsv1beta1.ExternalSecretDataRemoteRef{
								Key:            "some-key1",
								MetadataPolicy: externalsecretsv1beta1.ExternalSecretMetadataPolicyNone,
								Property:       "some-property1",
							},
						},
						{
							SecretKey: "ENVVAR2",
							RemoteRef: externalsecretsv1beta1.ExternalSecretDataRemoteRef{
								Key:            "some-key2",
								MetadataPolicy: externalsecretsv1beta1.ExternalSecretMetadataPolicyNone,
								Property:       "some-property2",
							},
						},
					},
				},
				desired: &externalsecretsv1beta1.ExternalSecretSpec{
					Data: []externalsecretsv1beta1.ExternalSecretData{
						{
							SecretKey: "ENVVAR1",
							RemoteRef: externalsecretsv1beta1.ExternalSecretDataRemoteRef{
								Key:            "some-key1",
								MetadataPolicy: "",
								Property:       "some-other-property1",
							},
						},
					},
				},
			},
			path: ".data[*].remoteRef.metadataPolicy",
			want: &ChangeSet[externalsecretsv1beta1.ExternalSecretSpec]{
				path: "spec",
				current: &externalsecretsv1beta1.ExternalSecretSpec{
					Data: []externalsecretsv1beta1.ExternalSecretData{
						{
							SecretKey: "ENVVAR1",
							RemoteRef: externalsecretsv1beta1.ExternalSecretDataRemoteRef{
								Key:      "some-key1",
								Property: "some-property1",
							},
						},
						{
							SecretKey: "ENVVAR2",
							RemoteRef: externalsecretsv1beta1.ExternalSecretDataRemoteRef{
								Key:      "some-key2",
								Property: "some-property2",
							},
						},
					},
				},
				desired: &externalsecretsv1beta1.ExternalSecretSpec{
					Data: []externalsecretsv1beta1.ExternalSecretData{
						{
							SecretKey: "ENVVAR1",
							RemoteRef: externalsecretsv1beta1.ExternalSecretDataRemoteRef{
								Key:      "some-key1",
								Property: "some-other-property1",
							},
						},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.changeset.removeMatchingProperties(tt.path); (err != nil) != tt.wantErr {
				t.Errorf("ChangeSet_removeMatchingProperties() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(tt.changeset.current, tt.want.current); len(diff) > 0 {
				t.Errorf("ChangeSet_removeMatchingProperties() diff in set.current = %v", diff)
			}
			if diff := cmp.Diff(tt.changeset.desired, tt.want.desired); len(diff) > 0 {
				t.Errorf("ChangeSet_removeMatchingProperties() diff in set.desired = %v", diff)
			}
		})
	}
}
