package google

import (
	"fmt"
	"log"
	"net"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	"google.golang.org/api/container/v1"
)

var (
	instanceGroupManagerURL = regexp.MustCompile("^https://www.googleapis.com/compute/v1/projects/([a-z][a-z0-9-]{5}(?:[-a-z0-9]{0,23}[a-z0-9])?)/zones/([a-z0-9-]*)/instanceGroupManagers/([^/]*)")
)

func resourceContainerCluster() *schema.Resource {
	return &schema.Resource{
		Create: resourceContainerClusterCreate,
		Read:   resourceContainerClusterRead,
		Update: resourceContainerClusterUpdate,
		Delete: resourceContainerClusterDelete,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(30 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		SchemaVersion: 1,
		MigrateState:  resourceContainerClusterMigrateState,

		Importer: &schema.ResourceImporter{
			State: resourceContainerClusterStateImporter,
		},

		Schema: map[string]*schema.Schema{
			"master_auth": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				MaxItems: 1,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"client_certificate": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"client_key": {
							Type:      schema.TypeString,
							Computed:  true,
							Sensitive: true,
						},
						"cluster_ca_certificate": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"password": {
							Type:      schema.TypeString,
							Required:  true,
							ForceNew:  true,
							Sensitive: true,
						},
						"username": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
					},
				},
			},

			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)

					if len(value) > 40 {
						errors = append(errors, fmt.Errorf(
							"%q cannot be longer than 40 characters", k))
					}
					if !regexp.MustCompile("^[a-z0-9-]+$").MatchString(value) {
						errors = append(errors, fmt.Errorf(
							"%q can only contain lowercase letters, numbers and hyphens", k))
					}
					if !regexp.MustCompile("^[a-z]").MatchString(value) {
						errors = append(errors, fmt.Errorf(
							"%q must start with a letter", k))
					}
					if !regexp.MustCompile("[a-z0-9]$").MatchString(value) {
						errors = append(errors, fmt.Errorf(
							"%q must end with a number or a letter", k))
					}
					return
				},
			},

			"zone": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"initial_node_count": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},

			"additional_zones": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"cluster_ipv4_cidr": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					_, ipnet, err := net.ParseCIDR(value)

					if err != nil || ipnet == nil || value != ipnet.String() {
						errors = append(errors, fmt.Errorf(
							"%q must contain a valid CIDR", k))
					}
					return
				},
			},

			"description": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"enable_legacy_abac": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},

			"endpoint": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"instance_group_urls": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"logging_service": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.StringInSlice([]string{"logging.googleapis.com", "none"}, false),
			},

			"monitoring_service": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"network": {
				Type:      schema.TypeString,
				Optional:  true,
				Default:   "default",
				ForceNew:  true,
				StateFunc: StoreResourceName,
			},
			"subnetwork": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"addons_config": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"http_load_balancing": {
							Type:     schema.TypeList,
							Optional: true,
							ForceNew: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"disabled": {
										Type:     schema.TypeBool,
										Optional: true,
										ForceNew: true,
									},
								},
							},
						},
						"horizontal_pod_autoscaling": {
							Type:     schema.TypeList,
							Optional: true,
							ForceNew: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"disabled": {
										Type:     schema.TypeBool,
										Optional: true,
										ForceNew: true,
									},
								},
							},
						},
						"kubernetes_dashboard": {
							Type:     schema.TypeList,
							Optional: true,
							ForceNew: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"disabled": {
										Type:     schema.TypeBool,
										Optional: true,
										ForceNew: true,
									},
								},
							},
						},
					},
				},
			},

			"master_version": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"node_config": schemaNodeConfig,

			"node_version": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"node_pool": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				ForceNew: true, // TODO(danawillow): Add ability to add/remove nodePools
				Elem: &schema.Resource{
					Schema: schemaNodePool,
				},
			},

			"project": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceContainerClusterCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	zoneName := d.Get("zone").(string)
	clusterName := d.Get("name").(string)

	cluster := &container.Cluster{
		Name:             clusterName,
		InitialNodeCount: int64(d.Get("initial_node_count").(int)),
	}

	timeoutInMinutes := int(d.Timeout(schema.TimeoutCreate).Minutes())

	if v, ok := d.GetOk("master_auth"); ok {
		masterAuths := v.([]interface{})
		masterAuth := masterAuths[0].(map[string]interface{})
		cluster.MasterAuth = &container.MasterAuth{
			Password: masterAuth["password"].(string),
			Username: masterAuth["username"].(string),
		}
	}

	if v, ok := d.GetOk("master_version"); ok {
		cluster.InitialClusterVersion = v.(string)
	}

	if _, ok := d.GetOk("node_version"); ok {
		return fmt.Errorf("cannot set node_version on create, use master_version instead")
	}

	if v, ok := d.GetOk("additional_zones"); ok {
		locationsList := v.(*schema.Set).List()
		locations := []string{}
		for _, v := range locationsList {
			location := v.(string)
			locations = append(locations, location)
			if location == zoneName {
				return fmt.Errorf("additional_zones should not contain the original 'zone'.")
			}
		}
		locations = append(locations, zoneName)
		cluster.Locations = locations
	}

	if v, ok := d.GetOk("cluster_ipv4_cidr"); ok {
		cluster.ClusterIpv4Cidr = v.(string)
	}

	if v, ok := d.GetOk("description"); ok {
		cluster.Description = v.(string)
	}

	cluster.LegacyAbac = &container.LegacyAbac{
		Enabled:         d.Get("enable_legacy_abac").(bool),
		ForceSendFields: []string{"Enabled"},
	}

	if v, ok := d.GetOk("logging_service"); ok {
		cluster.LoggingService = v.(string)
	}

	if v, ok := d.GetOk("monitoring_service"); ok {
		cluster.MonitoringService = v.(string)
	}

	if _, ok := d.GetOk("network"); ok {
		network, err := getNetworkName(d, "network")
		if err != nil {
			return err
		}
		cluster.Network = network
	}

	if v, ok := d.GetOk("subnetwork"); ok {
		cluster.Subnetwork = v.(string)
	}

	if v, ok := d.GetOk("addons_config"); ok {
		addonsConfig := v.([]interface{})[0].(map[string]interface{})
		cluster.AddonsConfig = &container.AddonsConfig{}

		if v, ok := addonsConfig["http_load_balancing"]; ok && len(v.([]interface{})) > 0 {
			addon := v.([]interface{})[0].(map[string]interface{})
			cluster.AddonsConfig.HttpLoadBalancing = &container.HttpLoadBalancing{
				Disabled: addon["disabled"].(bool),
			}
		}

		if v, ok := addonsConfig["horizontal_pod_autoscaling"]; ok && len(v.([]interface{})) > 0 {
			addon := v.([]interface{})[0].(map[string]interface{})
			cluster.AddonsConfig.HorizontalPodAutoscaling = &container.HorizontalPodAutoscaling{
				Disabled: addon["disabled"].(bool),
			}
		}

		if v, ok := addonsConfig["kubernetes_dashboard"]; ok && len(v.([]interface{})) > 0 {
			addon := v.([]interface{})[0].(map[string]interface{})
			cluster.AddonsConfig.KubernetesDashboard = &container.KubernetesDashboard{
				Disabled: addon["disabled"].(bool),
			}
		}
	}
	if v, ok := d.GetOk("node_config"); ok {
		cluster.NodeConfig = expandNodeConfig(v)
	}

	nodePoolsCount := d.Get("node_pool.#").(int)
	if nodePoolsCount > 0 {
		nodePools := make([]*container.NodePool, 0, nodePoolsCount)
		for i := 0; i < nodePoolsCount; i++ {
			prefix := fmt.Sprintf("node_pool.%d.", i)
			nodePool, err := expandNodePool(d, prefix)
			if err != nil {
				return err
			}
			nodePools = append(nodePools, nodePool)
		}
		cluster.NodePools = nodePools
	}

	req := &container.CreateClusterRequest{
		Cluster: cluster,
	}

	op, err := config.clientContainer.Projects.Zones.Clusters.Create(
		project, zoneName, req).Do()
	if err != nil {
		return err
	}

	// Wait until it's created
	waitErr := containerOperationWait(config, op, project, zoneName, "creating GKE cluster", timeoutInMinutes, 3)
	if waitErr != nil {
		// The resource didn't actually create
		d.SetId("")
		return waitErr
	}

	log.Printf("[INFO] GKE cluster %s has been created", clusterName)

	d.SetId(clusterName)

	return resourceContainerClusterRead(d, meta)
}

func resourceContainerClusterRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	zoneName := d.Get("zone").(string)

	cluster, err := config.clientContainer.Projects.Zones.Clusters.Get(
		project, zoneName, d.Get("name").(string)).Do()
	if err != nil {
		return handleNotFoundError(err, d, fmt.Sprintf("Container Cluster %q", d.Get("name").(string)))
	}

	d.Set("name", cluster.Name)
	d.Set("zone", cluster.Zone)

	locations := []string{}
	if len(cluster.Locations) > 1 {
		for _, location := range cluster.Locations {
			if location != cluster.Zone {
				locations = append(locations, location)
			}
		}
	}
	d.Set("additional_zones", locations)

	d.Set("endpoint", cluster.Endpoint)

	masterAuth := []map[string]interface{}{
		{
			"username":               cluster.MasterAuth.Username,
			"password":               cluster.MasterAuth.Password,
			"client_certificate":     cluster.MasterAuth.ClientCertificate,
			"client_key":             cluster.MasterAuth.ClientKey,
			"cluster_ca_certificate": cluster.MasterAuth.ClusterCaCertificate,
		},
	}
	d.Set("master_auth", masterAuth)

	d.Set("initial_node_count", cluster.InitialNodeCount)
	d.Set("master_version", cluster.CurrentMasterVersion)
	d.Set("node_version", cluster.CurrentNodeVersion)
	d.Set("cluster_ipv4_cidr", cluster.ClusterIpv4Cidr)
	d.Set("description", cluster.Description)
	d.Set("enable_legacy_abac", cluster.LegacyAbac.Enabled)
	d.Set("logging_service", cluster.LoggingService)
	d.Set("monitoring_service", cluster.MonitoringService)
	d.Set("network", cluster.Network)
	d.Set("subnetwork", cluster.Subnetwork)
	d.Set("node_config", flattenNodeConfig(cluster.NodeConfig))
	nps, err := flattenClusterNodePools(d, config, cluster.NodePools)
	if err != nil {
		return err
	}
	d.Set("node_pool", nps)

	if igUrls, err := getInstanceGroupUrlsFromManagerUrls(config, cluster.InstanceGroupUrls); err != nil {
		return err
	} else {
		d.Set("instance_group_urls", igUrls)
	}

	return nil
}

func resourceContainerClusterUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	zoneName := d.Get("zone").(string)
	clusterName := d.Get("name").(string)
	timeoutInMinutes := int(d.Timeout(schema.TimeoutUpdate).Minutes())

	d.Partial(true)

	// The master must be updated before the nodes
	if d.HasChange("master_version") {
		desiredMasterVersion := d.Get("master_version").(string)
		req := &container.UpdateClusterRequest{
			Update: &container.ClusterUpdate{
				DesiredMasterVersion: desiredMasterVersion,
			},
		}
		op, err := config.clientContainer.Projects.Zones.Clusters.Update(
			project, zoneName, clusterName, req).Do()
		if err != nil {
			return err
		}

		// Wait until it's updated
		waitErr := containerOperationWait(config, op, project, zoneName, "updating GKE master version", timeoutInMinutes, 2)
		if waitErr != nil {
			return waitErr
		}

		log.Printf("[INFO] GKE cluster %s: master has been updated to %s", d.Id(),
			desiredMasterVersion)

		d.SetPartial("master_version")
	}

	if d.HasChange("node_version") {
		desiredNodeVersion := d.Get("node_version").(string)

		req := &container.UpdateClusterRequest{
			Update: &container.ClusterUpdate{
				DesiredNodeVersion: desiredNodeVersion,
			},
		}
		op, err := config.clientContainer.Projects.Zones.Clusters.Update(
			project, zoneName, clusterName, req).Do()
		if err != nil {
			return err
		}

		// Wait until it's updated
		waitErr := containerOperationWait(config, op, project, zoneName, "updating GKE node version", timeoutInMinutes, 2)
		if waitErr != nil {
			return waitErr
		}

		log.Printf("[INFO] GKE cluster %s: nodes have been updated to %s", d.Id(),
			desiredNodeVersion)

		d.SetPartial("node_version")
	}

	if d.HasChange("additional_zones") {
		azSet := d.Get("additional_zones").(*schema.Set)
		if azSet.Contains(zoneName) {
			return fmt.Errorf("additional_zones should not contain the original 'zone'.")
		}
		azs := convertStringArr(azSet.List())
		locations := append(azs, zoneName)
		req := &container.UpdateClusterRequest{
			Update: &container.ClusterUpdate{
				DesiredLocations: locations,
			},
		}
		op, err := config.clientContainer.Projects.Zones.Clusters.Update(
			project, zoneName, clusterName, req).Do()
		if err != nil {
			return err
		}

		// Wait until it's updated
		waitErr := containerOperationWait(config, op, project, zoneName, "updating GKE cluster locations", timeoutInMinutes, 2)
		if waitErr != nil {
			return waitErr
		}

		log.Printf("[INFO] GKE cluster %s locations have been updated to %v", d.Id(),
			locations)

		d.SetPartial("additional_zones")
	}

	if d.HasChange("enable_legacy_abac") {
		enabled := d.Get("enable_legacy_abac").(bool)
		req := &container.SetLegacyAbacRequest{
			Enabled:         enabled,
			ForceSendFields: []string{"Enabled"},
		}
		op, err := config.clientContainer.Projects.Zones.Clusters.LegacyAbac(project, zoneName, clusterName, req).Do()
		if err != nil {
			return err
		}

		// Wait until it's updated
		waitErr := containerOperationWait(config, op, project, zoneName, "updating GKE legacy ABAC", timeoutInMinutes, 2)
		if waitErr != nil {
			return waitErr
		}

		log.Printf("[INFO] GKE cluster %s legacy ABAC has been updated to %v", d.Id(), enabled)

		d.SetPartial("enable_legacy_abac")
	}

	if n, ok := d.GetOk("node_pool.#"); ok {
		for i := 0; i < n.(int); i++ {
			if err := nodePoolUpdate(d, meta, clusterName, fmt.Sprintf("node_pool.%d.", i), timeoutInMinutes); err != nil {
				return err
			}
		}
		d.SetPartial("node_pool")
	}

	if d.HasChange("logging_service") {
		logging := d.Get("logging_service").(string)

		req := &container.SetLoggingServiceRequest{
			LoggingService: logging,
		}
		op, err := config.clientContainer.Projects.Zones.Clusters.Logging(
			project, zoneName, clusterName, req).Do()
		if err != nil {
			return err
		}

		// Wait until it's updated
		waitErr := containerOperationWait(config, op, project, zoneName, "updating GKE logging service", timeoutInMinutes, 2)
		if waitErr != nil {
			return waitErr
		}

		log.Printf("[INFO] GKE cluster %s: logging service has been updated to %s", d.Id(),
			logging)
		d.SetPartial("logging_service")
	}

	d.Partial(false)

	return resourceContainerClusterRead(d, meta)
}

func resourceContainerClusterDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	zoneName := d.Get("zone").(string)
	clusterName := d.Get("name").(string)
	timeoutInMinutes := int(d.Timeout(schema.TimeoutDelete).Minutes())

	log.Printf("[DEBUG] Deleting GKE cluster %s", d.Get("name").(string))
	op, err := config.clientContainer.Projects.Zones.Clusters.Delete(
		project, zoneName, clusterName).Do()
	if err != nil {
		return err
	}

	// Wait until it's deleted
	waitErr := containerOperationWait(config, op, project, zoneName, "deleting GKE cluster", timeoutInMinutes, 3)
	if waitErr != nil {
		return waitErr
	}

	log.Printf("[INFO] GKE cluster %s has been deleted", d.Id())

	d.SetId("")

	return nil
}

// container engine's API currently mistakenly returns the instance group manager's
// URL instead of the instance group's URL in its responses. This shim detects that
// error, and corrects it, by fetching the instance group manager URL and retrieving
// the instance group manager, then using that to look up the instance group URL, which
// is then substituted.
//
// This should be removed when the API response is fixed.
func getInstanceGroupUrlsFromManagerUrls(config *Config, igmUrls []string) ([]string, error) {
	instanceGroupURLs := make([]string, 0, len(igmUrls))
	for _, u := range igmUrls {
		if !instanceGroupManagerURL.MatchString(u) {
			instanceGroupURLs = append(instanceGroupURLs, u)
			continue
		}
		matches := instanceGroupManagerURL.FindStringSubmatch(u)
		instanceGroupManager, err := config.clientCompute.InstanceGroupManagers.Get(matches[1], matches[2], matches[3]).Do()
		if err != nil {
			return nil, fmt.Errorf("Error reading instance group manager returned as an instance group URL: %s", err)
		}
		instanceGroupURLs = append(instanceGroupURLs, instanceGroupManager.InstanceGroup)
	}
	return instanceGroupURLs, nil
}

func flattenClusterNodePools(d *schema.ResourceData, config *Config, c []*container.NodePool) ([]map[string]interface{}, error) {
	nodePools := make([]map[string]interface{}, 0, len(c))

	for i, np := range c {
		nodePool, err := flattenNodePool(d, config, np, fmt.Sprintf("node_pool.%d.", i))
		if err != nil {
			return nil, err
		}
		nodePools = append(nodePools, nodePool)
	}

	return nodePools, nil
}

func resourceContainerClusterStateImporter(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	parts := strings.Split(d.Id(), "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("Invalid container cluster specifier. Expecting {zone}/{name}")
	}

	d.Set("zone", parts[0])
	d.Set("name", parts[1])
	d.SetId(parts[1])

	return []*schema.ResourceData{d}, nil
}
