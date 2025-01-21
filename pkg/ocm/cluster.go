// This file defines a wrapper to the sdk's cluster object for easy mocking
//
//go:generate mockgen -source=cluster.go -package=ocm -destination=mock_cluster.go
package ocm

import (
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

type Cluster interface {
	Id() string
	State() cmv1.ClusterState
	CloudProviderId() string
	CcsEnabled() bool
}

type cluster struct {
	data *cmv1.Cluster
}

func NewCluster(data *cmv1.Cluster) Cluster {
	return &cluster{
		data: data,
	}
}

func (c *cluster) Id() string               { return c.data.ID() }
func (c *cluster) State() cmv1.ClusterState { return c.data.State() }
func (c *cluster) CloudProviderId() string  { return c.data.CloudProvider().ID() }
func (c *cluster) CcsEnabled() bool         { return c.data.CCS().Enabled() }
