package ping

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/stretchr/testify/require"
)

func Test_processRegions(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		regions  []*ps.Region
		rd       regionData
	}{
		{
			name: "empty",
			rd: regionData{
				Providers:     make(map[string]struct{}),
				RegionsBySlug: make(map[string]*ps.Region),
			},
		},
		{
			name: "all providers",
			regions: []*ps.Region{
				{
					Slug:     "aws-us-east-2",
					Provider: "AWS",
				},
				{
					Slug:     "gcp-us-east4",
					Provider: "GCP",
				},
			},
			rd: regionData{
				Providers: map[string]struct{}{
					"aws": {},
					"gcp": {},
				},
				RegionsBySlug: map[string]*ps.Region{
					"aws-us-east-2": {
						Slug:     "aws-us-east-2",
						Provider: "AWS",
					},
					"gcp-us-east4": {
						Slug:     "gcp-us-east4",
						Provider: "GCP",
					},
				},
				Endpoints: []string{
					"aws",
					"aws-us-east-2",
					"gcp",
					"gcp-us-east4",
				},
			},
		},
		{
			name:     "AWS",
			provider: "AWS",
			regions: []*ps.Region{
				{
					Slug:     "aws-us-east-2",
					Provider: "AWS",
				},
				{
					Slug:     "gcp-us-east4",
					Provider: "GCP",
				},
			},
			rd: regionData{
				Providers: map[string]struct{}{
					"aws": {},
					"gcp": {},
				},
				RegionsBySlug: map[string]*ps.Region{
					"aws-us-east-2": {
						Slug:     "aws-us-east-2",
						Provider: "AWS",
					},
					"gcp-us-east4": {
						Slug:     "gcp-us-east4",
						Provider: "GCP",
					},
				},
				Endpoints: []string{
					"aws",
					"aws-us-east-2",
				},
			},
		},
		{
			name:     "GCP",
			provider: "GCP",
			regions: []*ps.Region{
				{
					Slug:     "aws-us-east-2",
					Provider: "AWS",
				},
				{
					Slug:     "gcp-us-east4",
					Provider: "GCP",
				},
			},
			rd: regionData{
				Providers: map[string]struct{}{
					"aws": {},
					"gcp": {},
				},
				RegionsBySlug: map[string]*ps.Region{
					"aws-us-east-2": {
						Slug:     "aws-us-east-2",
						Provider: "AWS",
					},
					"gcp-us-east4": {
						Slug:     "gcp-us-east4",
						Provider: "GCP",
					},
				},
				Endpoints: []string{
					"gcp",
					"gcp-us-east4",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Empty(t, cmp.Diff(tt.rd, processRegions(tt.provider, tt.regions)))
		})
	}
}
