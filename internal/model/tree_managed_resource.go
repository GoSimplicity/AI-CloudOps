package model

// AliyunRDSInstance is a read-only metadata projection, not an ECS resource.
type AliyunRDSInstance struct {
	InstanceID    string `json:"instance_id"`
	Name          string `json:"name"`
	Region        string `json:"region"`
	Zone          string `json:"zone"`
	Status        string `json:"status"`
	Engine        string `json:"engine"`
	EngineVersion string `json:"engine_version"`
	InstanceClass string `json:"instance_class"`
}

// AliyunLoadBalancer is the classic SLB service, not ALB/NLB.
type AliyunLoadBalancer struct {
	InstanceID  string `json:"instance_id"`
	Name        string `json:"name"`
	Region      string `json:"region"`
	Status      string `json:"status"`
	AddressType string `json:"address_type"`
	VPCID       string `json:"vpc_id"`
}

// AliyunCacheInstance represents r-kvstore metadata; secrets/endpoints are excluded.
type AliyunCacheInstance struct {
	InstanceID    string `json:"instance_id"`
	Name          string `json:"name"`
	Region        string `json:"region"`
	Zone          string `json:"zone"`
	Status        string `json:"status"`
	InstanceType  string `json:"instance_type"`
	EngineVersion string `json:"engine_version"`
	InstanceClass string `json:"instance_class"`
}
