// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package mq

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/YakDriver/regexache"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/mq"
	"github.com/aws/aws-sdk-go-v2/service/mq/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/id"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/create"
	"github.com/hashicorp/terraform-provider-aws/internal/enum"
	"github.com/hashicorp/terraform-provider-aws/internal/errs"
	"github.com/hashicorp/terraform-provider-aws/internal/errs/sdkdiag"
	"github.com/hashicorp/terraform-provider-aws/internal/flex"
	tftags "github.com/hashicorp/terraform-provider-aws/internal/tags"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
	"github.com/hashicorp/terraform-provider-aws/internal/types/nullable"
	"github.com/hashicorp/terraform-provider-aws/internal/verify"
	"github.com/hashicorp/terraform-provider-aws/names"
	"github.com/mitchellh/copystructure"
)

// @SDKResource("aws_mq_broker", name="Broker")
// @Tags(identifierAttribute="arn")
func resourceBroker() *schema.Resource {
	return &schema.Resource{
		CreateWithoutTimeout: resourceBrokerCreate,
		ReadWithoutTimeout:   resourceBrokerRead,
		UpdateWithoutTimeout: resourceBrokerUpdate,
		DeleteWithoutTimeout: resourceBrokerDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(30 * time.Minute),
			Update: schema.DefaultTimeout(30 * time.Minute),
			Delete: schema.DefaultTimeout(30 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"apply_immediately": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"authentication_strategy": {
				Type:             schema.TypeString,
				Optional:         true,
				Computed:         true,
				ValidateDiagFunc: enum.ValidateIgnoreCase[types.AuthenticationStrategy](),
			},
			"auto_minor_version_upgrade": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"broker_name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: ValidateBrokerName,
			},
			"configuration": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
						"revision": {
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
						},
					},
				},
			},
			"deployment_mode": {
				Type:             schema.TypeString,
				Optional:         true,
				ForceNew:         true,
				Default:          types.DeploymentModeSingleInstance,
				ValidateDiagFunc: enum.ValidateIgnoreCase[types.DeploymentMode](),
			},
			"encryption_options": {
				Type:             schema.TypeList,
				Optional:         true,
				ForceNew:         true,
				MaxItems:         1,
				DiffSuppressFunc: verify.SuppressMissingOptionalConfigurationBlock,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"kms_key_id": {
							Type:         schema.TypeString,
							Optional:     true,
							Computed:     true,
							ForceNew:     true,
							ValidateFunc: verify.ValidARN,
						},
						"use_aws_owned_key": {
							Type:     schema.TypeBool,
							Optional: true,
							ForceNew: true,
							Default:  true,
						},
					},
				},
			},
			"engine_type": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: enum.ValidateIgnoreCase[types.EngineType](),
			},
			"engine_version": {
				Type:     schema.TypeString,
				Required: true,
			},
			"host_instance_type": {
				Type:     schema.TypeString,
				Required: true,
			},
			"instances": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"console_url": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"endpoints": {
							Type:     schema.TypeList,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"ip_address": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"ldap_server_metadata": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"hosts": {
							Type:     schema.TypeList,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"role_base": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"role_name": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"role_search_matching": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"role_search_subtree": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"service_account_password": {
							Type:      schema.TypeString,
							Optional:  true,
							Sensitive: true,
						},
						"service_account_username": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"user_base": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"user_role_name": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"user_search_matching": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"user_search_subtree": {
							Type:     schema.TypeBool,
							Optional: true,
						},
					},
				},
			},
			"logs": {
				Type:             schema.TypeList,
				Optional:         true,
				MaxItems:         1,
				DiffSuppressFunc: verify.SuppressMissingOptionalConfigurationBlock,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"audit": {
							Type:             nullable.TypeNullableBool,
							Optional:         true,
							ValidateFunc:     nullable.ValidateTypeStringNullableBool,
							DiffSuppressFunc: nullable.DiffSuppressNullableBoolFalseAsNull,
						},
						"general": {
							Type:     schema.TypeBool,
							Optional: true,
						},
					},
				},
			},
			"maintenance_window_start_time": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"day_of_week": {
							Type:             schema.TypeString,
							Required:         true,
							ValidateDiagFunc: enum.ValidateIgnoreCase[types.DayOfWeek](),
						},
						"time_of_day": {
							Type:     schema.TypeString,
							Required: true,
						},
						"time_zone": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"publicly_accessible": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
				Default:  false,
			},
			"security_groups": {
				Type:     schema.TypeSet,
				Optional: true,
				MaxItems: 5,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"storage_type": {
				Type:             schema.TypeString,
				Optional:         true,
				Computed:         true,
				ValidateDiagFunc: enum.ValidateIgnoreCase[types.BrokerStorageType](),
			},
			"subnet_ids": {
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			names.AttrTags:    tftags.TagsSchema(),
			names.AttrTagsAll: tftags.TagsSchemaComputed(),
			"user": {
				Type:     schema.TypeSet,
				Required: true,
				Set:      resourceUserHash,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					// AWS currently does not support updating the RabbitMQ users beyond resource creation.
					// User list is not returned back after creation.
					// Updates to users can only be in the RabbitMQ UI.
					if v := d.Get("engine_type").(string); strings.EqualFold(v, string(types.EngineTypeRabbitmq)) && d.Get("arn").(string) != "" {
						return true
					}

					return false
				},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"console_access": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"groups": {
							Type:     schema.TypeSet,
							Optional: true,
							MaxItems: 20,
							Elem: &schema.Schema{
								Type:         schema.TypeString,
								ValidateFunc: validation.StringLenBetween(2, 100),
							},
						},
						"password": {
							Type:         schema.TypeString,
							Required:     true,
							Sensitive:    true,
							ValidateFunc: ValidBrokerPassword,
						},
						"replication_user": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"username": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringLenBetween(2, 100),
						},
					},
				},
			},
		},

		CustomizeDiff: customdiff.All(
			verify.SetTagsDiff,
			func(_ context.Context, diff *schema.ResourceDiff, v interface{}) error {
				if strings.EqualFold(diff.Get("engine_type").(string), string(types.EngineTypeRabbitmq)) {
					if v, ok := diff.GetOk("logs.0.audit"); ok {
						if v, _, _ := nullable.Bool(v.(string)).Value(); v {
							return errors.New("logs.audit: Can not be configured when engine is RabbitMQ")
						}
					}
				}

				return nil
			},
		),
	}
}

func resourceBrokerCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	conn := meta.(*conns.AWSClient).MQClient(ctx)

	name := d.Get("broker_name").(string)
	engineType := d.Get("engine_type").(string)
	input := &mq.CreateBrokerInput{
		AutoMinorVersionUpgrade: aws.Bool(d.Get("auto_minor_version_upgrade").(bool)),
		BrokerName:              aws.String(name),
		CreatorRequestId:        aws.String(id.PrefixedUniqueId(fmt.Sprintf("tf-%s", name))),
		EngineType:              types.EngineType(engineType),
		EngineVersion:           aws.String(d.Get("engine_version").(string)),
		HostInstanceType:        aws.String(d.Get("host_instance_type").(string)),
		PubliclyAccessible:      aws.Bool(d.Get("publicly_accessible").(bool)),
		Tags:                    getTagsIn(ctx),
		Users:                   expandUsers(d.Get("user").(*schema.Set).List()),
	}

	if v, ok := d.GetOk("authentication_strategy"); ok {
		input.AuthenticationStrategy = types.AuthenticationStrategy(v.(string))
	}
	if v, ok := d.GetOk("configuration"); ok && len(v.([]interface{})) > 0 && v.([]interface{})[0] != nil {
		input.Configuration = expandConfigurationId(v.([]interface{}))
	}
	if v, ok := d.GetOk("deployment_mode"); ok {
		input.DeploymentMode = types.DeploymentMode(v.(string))
	}
	if v, ok := d.GetOk("encryption_options"); ok && len(v.([]interface{})) > 0 && v.([]interface{})[0] != nil {
		input.EncryptionOptions = expandEncryptionOptions(d.Get("encryption_options").([]interface{}))
	}
	if v, ok := d.GetOk("ldap_server_metadata"); ok && len(v.([]interface{})) > 0 && v.([]interface{})[0] != nil {
		input.LdapServerMetadata = expandLDAPServerMetadata(v.([]interface{}))
	}
	if v, ok := d.GetOk("logs"); ok && len(v.([]interface{})) > 0 && v.([]interface{})[0] != nil {
		input.Logs = expandLogs(engineType, v.([]interface{}))
	}
	if v, ok := d.GetOk("maintenance_window_start_time"); ok && len(v.([]interface{})) > 0 && v.([]interface{})[0] != nil {
		input.MaintenanceWindowStartTime = expandWeeklyStartTime(v.([]interface{}))
	}
	if v, ok := d.GetOk("security_groups"); ok && v.(*schema.Set).Len() > 0 {
		input.SecurityGroups = flex.ExpandStringValueSet(v.(*schema.Set))
	}
	if v, ok := d.GetOk("storage_type"); ok {
		input.StorageType = types.BrokerStorageType(v.(string))
	}
	if v, ok := d.GetOk("subnet_ids"); ok {
		input.SubnetIds = flex.ExpandStringValueSet(v.(*schema.Set))
	}

	output, err := conn.CreateBroker(ctx, input)

	if err != nil {
		return sdkdiag.AppendErrorf(diags, "creating MQ Broker (%s): %s", name, err)
	}

	d.SetId(aws.ToString(output.BrokerId))
	d.Set("arn", output.BrokerArn)

	if _, err := waitBrokerCreated(ctx, conn, d.Id(), d.Timeout(schema.TimeoutCreate)); err != nil {
		return sdkdiag.AppendErrorf(diags, "waiting for MQ Broker (%s) create: %s", d.Id(), err)
	}

	return append(diags, resourceBrokerRead(ctx, d, meta)...)
}

func resourceBrokerRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	conn := meta.(*conns.AWSClient).MQClient(ctx)

	output, err := findBrokerByID(ctx, conn, d.Id())

	if !d.IsNewResource() && (tfresource.NotFound(err) || errs.IsA[*types.ForbiddenException](err)) {
		log.Printf("[WARN] MQ Broker (%s) not found, removing from state", d.Id())
		d.SetId("")
		return diags
	}

	if err != nil {
		return sdkdiag.AppendErrorf(diags, "reading MQ Broker (%s): %s", d.Id(), err)
	}

	d.Set("arn", output.BrokerArn)
	d.Set("authentication_strategy", output.AuthenticationStrategy)
	d.Set("auto_minor_version_upgrade", output.AutoMinorVersionUpgrade)
	d.Set("broker_name", output.BrokerName)
	d.Set("deployment_mode", output.DeploymentMode)
	d.Set("engine_type", output.EngineType)
	d.Set("engine_version", output.EngineVersion)
	d.Set("host_instance_type", output.HostInstanceType)
	d.Set("instances", flattenBrokerInstances(output.BrokerInstances))
	d.Set("publicly_accessible", output.PubliclyAccessible)
	d.Set("security_groups", output.SecurityGroups)
	d.Set("storage_type", output.StorageType)
	d.Set("subnet_ids", output.SubnetIds)

	if err := d.Set("configuration", flattenConfiguration(output.Configurations)); err != nil {
		return sdkdiag.AppendErrorf(diags, "setting configuration: %s", err)
	}

	if err := d.Set("encryption_options", flattenEncryptionOptions(output.EncryptionOptions)); err != nil {
		return sdkdiag.AppendErrorf(diags, "setting encryption_options: %s", err)
	}

	var password string
	if v, ok := d.GetOk("ldap_server_metadata.0.service_account_password"); ok {
		password = v.(string)
	}

	if err := d.Set("ldap_server_metadata", flattenLDAPServerMetadata(output.LdapServerMetadata, password)); err != nil {
		return sdkdiag.AppendErrorf(diags, "setting ldap_server_metadata: %s", err)
	}

	if err := d.Set("logs", flattenLogs(output.Logs)); err != nil {
		return sdkdiag.AppendErrorf(diags, "setting logs: %s", err)
	}

	if err := d.Set("maintenance_window_start_time", flattenWeeklyStartTime(output.MaintenanceWindowStartTime)); err != nil {
		return sdkdiag.AppendErrorf(diags, "setting maintenance_window_start_time: %s", err)
	}

	rawUsers, err := expandUsersForBroker(ctx, conn, d.Id(), output.Users)

	if err != nil {
		return sdkdiag.AppendErrorf(diags, "reading MQ Broker (%s) users: %s", d.Id(), err)
	}

	if err := d.Set("user", flattenUsers(rawUsers, d.Get("user").(*schema.Set).List())); err != nil {
		return sdkdiag.AppendErrorf(diags, "setting user: %s", err)
	}

	setTagsOut(ctx, output.Tags)

	return diags
}

func resourceBrokerUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	conn := meta.(*conns.AWSClient).MQClient(ctx)

	requiresReboot := false

	if d.HasChange("security_groups") {
		input := &mq.UpdateBrokerInput{
			BrokerId:       aws.String(d.Id()),
			SecurityGroups: flex.ExpandStringValueSet(d.Get("security_groups").(*schema.Set)),
		}

		_, err := conn.UpdateBroker(ctx, input)

		if err != nil {
			return sdkdiag.AppendErrorf(diags, "updating MQ Broker (%s) security groups: %s", d.Id(), err)
		}
	}

	if d.HasChanges("configuration", "logs", "engine_version") {
		input := &mq.UpdateBrokerInput{
			BrokerId:      aws.String(d.Id()),
			Configuration: expandConfigurationId(d.Get("configuration").([]interface{})),
			EngineVersion: aws.String(d.Get("engine_version").(string)),
			Logs:          expandLogs(d.Get("engine_type").(string), d.Get("logs").([]interface{})),
		}

		_, err := conn.UpdateBroker(ctx, input)

		if err != nil {
			return sdkdiag.AppendErrorf(diags, "updating MQ Broker (%s) configuration: %s", d.Id(), err)
		}

		requiresReboot = true
	}

	if d.HasChange("user") {
		o, n := d.GetChange("user")
		var err error
		// d.HasChange("user") always reports a change when running resourceBrokerUpdate
		// updateBrokerUsers needs to be called to know if changes to user are actually made
		var usersUpdated bool
		usersUpdated, err = updateBrokerUsers(ctx, conn, d.Id(), o.(*schema.Set).List(), n.(*schema.Set).List())

		if err != nil {
			return sdkdiag.AppendErrorf(diags, "updating MQ Broker (%s) users: %s", d.Id(), err)
		}

		if usersUpdated {
			requiresReboot = true
		}
	}

	if d.HasChange("host_instance_type") {
		input := &mq.UpdateBrokerInput{
			BrokerId:         aws.String(d.Id()),
			HostInstanceType: aws.String(d.Get("host_instance_type").(string)),
		}

		_, err := conn.UpdateBroker(ctx, input)

		if err != nil {
			return sdkdiag.AppendErrorf(diags, "updating MQ Broker (%s) host instance type: %s", d.Id(), err)
		}

		requiresReboot = true
	}

	if d.HasChange("auto_minor_version_upgrade") {
		input := &mq.UpdateBrokerInput{
			AutoMinorVersionUpgrade: aws.Bool(d.Get("auto_minor_version_upgrade").(bool)),
			BrokerId:                aws.String(d.Id()),
		}

		_, err := conn.UpdateBroker(ctx, input)

		if err != nil {
			return sdkdiag.AppendErrorf(diags, "updating MQ Broker (%s) auto minor version upgrade: %s", d.Id(), err)
		}

		requiresReboot = true
	}

	if d.HasChange("maintenance_window_start_time") {
		input := &mq.UpdateBrokerInput{
			BrokerId:                   aws.String(d.Id()),
			MaintenanceWindowStartTime: expandWeeklyStartTime(d.Get("maintenance_window_start_time").([]interface{})),
		}

		_, err := conn.UpdateBroker(ctx, input)

		if err != nil {
			return sdkdiag.AppendErrorf(diags, "updating MQ Broker (%s) maintenance window start time: %s", d.Id(), err)
		}

		requiresReboot = true
	}

	if d.Get("apply_immediately").(bool) && requiresReboot {
		_, err := conn.RebootBroker(ctx, &mq.RebootBrokerInput{
			BrokerId: aws.String(d.Id()),
		})

		if err != nil {
			return sdkdiag.AppendErrorf(diags, "rebooting MQ Broker (%s): %s", d.Id(), err)
		}

		if _, err := waitBrokerRebooted(ctx, conn, d.Id(), d.Timeout(schema.TimeoutUpdate)); err != nil {
			return sdkdiag.AppendErrorf(diags, "waiting for MQ Broker (%s) reboot: %s", d.Id(), err)
		}
	}

	return diags
}

func resourceBrokerDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	conn := meta.(*conns.AWSClient).MQClient(ctx)

	log.Printf("[INFO] Deleting MQ Broker: %s", d.Id())
	_, err := conn.DeleteBroker(ctx, &mq.DeleteBrokerInput{
		BrokerId: aws.String(d.Id()),
	})

	if errs.IsA[*types.NotFoundException](err) {
		return diags
	}

	if err != nil {
		return sdkdiag.AppendErrorf(diags, "deleting MQ Broker (%s): %s", d.Id(), err)
	}

	if _, err := waitBrokerDeleted(ctx, conn, d.Id(), d.Timeout(schema.TimeoutDelete)); err != nil {
		return sdkdiag.AppendErrorf(diags, "waiting for MQ Broker (%s) delete: %s", d.Id(), err)
	}

	return diags
}

func findBrokerByID(ctx context.Context, conn *mq.Client, id string) (*mq.DescribeBrokerOutput, error) {
	input := &mq.DescribeBrokerInput{
		BrokerId: aws.String(id),
	}

	output, err := conn.DescribeBroker(ctx, input)

	if errs.IsA[*types.NotFoundException](err) {
		return nil, &retry.NotFoundError{
			LastError:   err,
			LastRequest: input,
		}
	}

	if err != nil {
		return nil, err
	}

	if output == nil {
		return nil, tfresource.NewEmptyResultError(input)
	}

	return output, nil
}

func statusBrokerState(ctx context.Context, conn *mq.Client, id string) retry.StateRefreshFunc {
	return func() (interface{}, string, error) {
		output, err := findBrokerByID(ctx, conn, id)

		if tfresource.NotFound(err) {
			return nil, "", nil
		}

		if err != nil {
			return nil, "", err
		}

		return output, string(output.BrokerState), nil
	}
}

func waitBrokerCreated(ctx context.Context, conn *mq.Client, id string, timeout time.Duration) (*mq.DescribeBrokerOutput, error) {
	stateConf := retry.StateChangeConf{
		Pending: enum.Slice(types.BrokerStateCreationInProgress, types.BrokerStateRebootInProgress),
		Target:  enum.Slice(types.BrokerStateRunning),
		Timeout: timeout,
		Refresh: statusBrokerState(ctx, conn, id),
	}
	outputRaw, err := stateConf.WaitForStateContext(ctx)

	if output, ok := outputRaw.(*mq.DescribeBrokerOutput); ok {
		return output, err
	}

	return nil, err
}

func waitBrokerDeleted(ctx context.Context, conn *mq.Client, id string, timeout time.Duration) (*mq.DescribeBrokerOutput, error) {
	stateConf := retry.StateChangeConf{
		Pending: enum.Slice(
			types.BrokerStateCreationFailed,
			types.BrokerStateDeletionInProgress,
			types.BrokerStateRebootInProgress,
			types.BrokerStateRunning,
		),
		Target:  []string{},
		Timeout: timeout,
		Refresh: statusBrokerState(ctx, conn, id),
	}
	outputRaw, err := stateConf.WaitForStateContext(ctx)

	if output, ok := outputRaw.(*mq.DescribeBrokerOutput); ok {
		return output, err
	}

	return nil, err
}

func waitBrokerRebooted(ctx context.Context, conn *mq.Client, id string, timeout time.Duration) (*mq.DescribeBrokerOutput, error) {
	stateConf := retry.StateChangeConf{
		Pending: enum.Slice(types.BrokerStateRebootInProgress),
		Target:  enum.Slice(types.BrokerStateRunning),
		Timeout: timeout,
		Refresh: statusBrokerState(ctx, conn, id),
	}
	outputRaw, err := stateConf.WaitForStateContext(ctx)

	if output, ok := outputRaw.(*mq.DescribeBrokerOutput); ok {
		return output, err
	}

	return nil, err
}

func resourceUserHash(v interface{}) int {
	var buf bytes.Buffer

	m := v.(map[string]interface{})
	if ca, ok := m["console_access"]; ok {
		buf.WriteString(fmt.Sprintf("%t-", ca.(bool)))
	} else {
		buf.WriteString("false-")
	}
	if g, ok := m["groups"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", g.([]string)))
	}
	if p, ok := m["password"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", p.(string)))
	}
	buf.WriteString(fmt.Sprintf("%s-", m["username"].(string)))

	return create.StringHashcode(buf.String())
}

func updateBrokerUsers(ctx context.Context, conn *mq.Client, id string, oldUsers, newUsers []interface{}) (bool, error) {
	// If there are any user creates/deletes/updates, updatedUsers will be set to true
	updatedUsers := false

	createL, deleteL, updateL, err := DiffBrokerUsers(id, oldUsers, newUsers)
	if err != nil {
		return updatedUsers, err
	}

	for _, c := range createL {
		_, err := conn.CreateUser(ctx, c)
		updatedUsers = true
		if err != nil {
			return updatedUsers, err
		}
	}
	for _, d := range deleteL {
		_, err := conn.DeleteUser(ctx, d)
		updatedUsers = true
		if err != nil {
			return updatedUsers, err
		}
	}
	for _, u := range updateL {
		_, err := conn.UpdateUser(ctx, u)
		updatedUsers = true
		if err != nil {
			return updatedUsers, err
		}
	}

	return updatedUsers, nil
}

func DiffBrokerUsers(bId string, oldUsers, newUsers []interface{}) (cr []*mq.CreateUserInput, di []*mq.DeleteUserInput, ur []*mq.UpdateUserInput, e error) {
	existingUsers := make(map[string]interface{})
	for _, ou := range oldUsers {
		u := ou.(map[string]interface{})
		username := u["username"].(string)
		// Convert Set to slice to allow easier comparison
		if g, ok := u["groups"]; ok {
			groups := g.(*schema.Set).List()
			u["groups"] = groups
		}

		existingUsers[username] = u
	}

	for _, nu := range newUsers {
		// Still need access to the original map
		// because Set contents doesn't get copied
		// Likely related to https://github.com/mitchellh/copystructure/issues/17
		nuOriginal := nu.(map[string]interface{})

		// Create a mutable copy
		newUser, err := copystructure.Copy(nu)
		if err != nil {
			return cr, di, ur, err
		}

		newUserMap := newUser.(map[string]interface{})
		username := newUserMap["username"].(string)

		// Convert Set to slice to allow easier comparison
		var ng []interface{}
		if g, ok := nuOriginal["groups"]; ok {
			ng = g.(*schema.Set).List()
			newUserMap["groups"] = ng
		}

		if eu, ok := existingUsers[username]; ok {
			existingUserMap := eu.(map[string]interface{})

			if !reflect.DeepEqual(existingUserMap, newUserMap) {
				ur = append(ur, &mq.UpdateUserInput{
					BrokerId:        aws.String(bId),
					ConsoleAccess:   aws.Bool(newUserMap["console_access"].(bool)),
					Groups:          flex.ExpandStringValueList(ng),
					ReplicationUser: aws.Bool(newUserMap["replication_user"].(bool)),
					Password:        aws.String(newUserMap["password"].(string)),
					Username:        aws.String(username),
				})
			}

			// Delete after processing, so we know what's left for deletion
			delete(existingUsers, username)
		} else {
			cur := &mq.CreateUserInput{
				BrokerId:        aws.String(bId),
				ConsoleAccess:   aws.Bool(newUserMap["console_access"].(bool)),
				Password:        aws.String(newUserMap["password"].(string)),
				ReplicationUser: aws.Bool(newUserMap["replication_user"].(bool)),
				Username:        aws.String(username),
			}
			if len(ng) > 0 {
				cur.Groups = flex.ExpandStringValueList(ng)
			}
			cr = append(cr, cur)
		}
	}

	for username := range existingUsers {
		di = append(di, &mq.DeleteUserInput{
			BrokerId: aws.String(bId),
			Username: aws.String(username),
		})
	}

	return cr, di, ur, nil
}

func expandEncryptionOptions(l []interface{}) *types.EncryptionOptions {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	m := l[0].(map[string]interface{})

	encryptionOptions := &types.EncryptionOptions{
		UseAwsOwnedKey: aws.Bool(m["use_aws_owned_key"].(bool)),
	}

	if v, ok := m["kms_key_id"].(string); ok && v != "" {
		encryptionOptions.KmsKeyId = aws.String(v)
	}

	return encryptionOptions
}

func flattenEncryptionOptions(encryptionOptions *types.EncryptionOptions) []interface{} {
	if encryptionOptions == nil {
		return []interface{}{}
	}

	m := map[string]interface{}{
		"kms_key_id":        aws.ToString(encryptionOptions.KmsKeyId),
		"use_aws_owned_key": aws.ToBool(encryptionOptions.UseAwsOwnedKey),
	}

	return []interface{}{m}
}

func ValidBrokerPassword(v interface{}, k string) (ws []string, errors []error) {
	min := 12
	max := 250
	value := v.(string)
	unique := make(map[string]bool)

	for _, v := range value {
		if _, ok := unique[string(v)]; ok {
			continue
		}
		if string(v) == "," {
			errors = append(errors, fmt.Errorf("%q must not contain commas", k))
		}
		unique[string(v)] = true
	}
	if len(unique) < 4 {
		errors = append(errors, fmt.Errorf("%q must contain at least 4 unique characters", k))
	}
	if len(value) < min || len(value) > max {
		errors = append(errors, fmt.Errorf(
			"%q must be %d to %d characters long. provided string length: %d", k, min, max, len(value)))
	}
	return
}

func expandUsers(cfg []interface{}) []types.User {
	users := make([]types.User, len(cfg))
	for i, m := range cfg {
		u := m.(map[string]interface{})
		user := types.User{
			Username: aws.String(u["username"].(string)),
			Password: aws.String(u["password"].(string)),
		}
		if v, ok := u["console_access"]; ok {
			user.ConsoleAccess = aws.Bool(v.(bool))
		}
		if v, ok := u["replication_user"]; ok {
			user.ReplicationUser = aws.Bool(v.(bool))
		}
		if v, ok := u["groups"]; ok {
			user.Groups = flex.ExpandStringValueSet(v.(*schema.Set))
		}
		users[i] = user
	}
	return users
}

func expandUsersForBroker(ctx context.Context, conn *mq.Client, brokerId string, input []types.UserSummary) ([]*types.User, error) {
	var rawUsers []*types.User

	for _, u := range input {
		uOut, err := conn.DescribeUser(ctx, &mq.DescribeUserInput{
			BrokerId: aws.String(brokerId),
			Username: u.Username,
		})

		if err != nil {
			return nil, err
		}

		user := &types.User{
			ConsoleAccess:   uOut.ConsoleAccess,
			Groups:          uOut.Groups,
			ReplicationUser: uOut.ReplicationUser,
			Username:        uOut.Username,
		}

		rawUsers = append(rawUsers, user)
	}

	return rawUsers, nil
}

// We use cfgdUsers to get & set the password
func flattenUsers(users []*types.User, cfgUsers []interface{}) *schema.Set {
	existingPairs := make(map[string]string)
	for _, u := range cfgUsers {
		user := u.(map[string]interface{})
		username := user["username"].(string)
		existingPairs[username] = user["password"].(string)
	}

	out := make([]interface{}, 0)
	for _, u := range users {
		m := map[string]interface{}{
			"username": aws.ToString(u.Username),
		}
		password := ""
		if p, ok := existingPairs[aws.ToString(u.Username)]; ok {
			password = p
		}
		if password != "" {
			m["password"] = password
		}
		if u.ConsoleAccess != nil {
			m["console_access"] = aws.ToBool(u.ConsoleAccess)
		}
		if u.ReplicationUser != nil {
			m["replication_user"] = aws.ToBool(u.ReplicationUser)
		}
		if len(u.Groups) > 0 {
			m["groups"] = u.Groups
		}
		out = append(out, m)
	}
	return schema.NewSet(resourceUserHash, out)
}

func expandWeeklyStartTime(cfg []interface{}) *types.WeeklyStartTime {
	if len(cfg) < 1 {
		return nil
	}

	m := cfg[0].(map[string]interface{})
	return &types.WeeklyStartTime{
		DayOfWeek: types.DayOfWeek(m["day_of_week"].(string)),
		TimeOfDay: aws.String(m["time_of_day"].(string)),
		TimeZone:  aws.String(m["time_zone"].(string)),
	}
}

func flattenWeeklyStartTime(wst *types.WeeklyStartTime) []interface{} {
	if wst == nil {
		return []interface{}{}
	}
	m := make(map[string]interface{})
	if wst.DayOfWeek != "" {
		m["day_of_week"] = wst.DayOfWeek
	}
	if wst.TimeOfDay != nil {
		m["time_of_day"] = aws.ToString(wst.TimeOfDay)
	}
	if wst.TimeZone != nil {
		m["time_zone"] = aws.ToString(wst.TimeZone)
	}
	return []interface{}{m}
}

func expandConfigurationId(cfg []interface{}) *types.ConfigurationId {
	if len(cfg) < 1 {
		return nil
	}

	m := cfg[0].(map[string]interface{})
	out := types.ConfigurationId{
		Id: aws.String(m["id"].(string)),
	}
	if v, ok := m["revision"].(int); ok && v > 0 {
		out.Revision = aws.Int32(int32(v))
	}

	return &out
}

func flattenConfiguration(config *types.Configurations) []interface{} {
	if config == nil || config.Current == nil {
		return []interface{}{}
	}

	m := map[string]interface{}{
		"id":       aws.ToString(config.Current.Id),
		"revision": aws.ToInt32(config.Current.Revision),
	}

	return []interface{}{m}
}

func flattenBrokerInstances(instances []types.BrokerInstance) []interface{} {
	if len(instances) == 0 {
		return []interface{}{}
	}
	l := make([]interface{}, len(instances))
	for i, instance := range instances {
		m := make(map[string]interface{})
		if instance.ConsoleURL != nil {
			m["console_url"] = aws.ToString(instance.ConsoleURL)
		}
		if len(instance.Endpoints) > 0 {
			m["endpoints"] = instance.Endpoints
		}
		if instance.IpAddress != nil {
			m["ip_address"] = aws.ToString(instance.IpAddress)
		}
		l[i] = m
	}

	return l
}

func flattenLogs(logs *types.LogsSummary) []interface{} {
	if logs == nil {
		return []interface{}{}
	}

	m := map[string]interface{}{}

	if logs.General != nil {
		m["general"] = aws.ToBool(logs.General)
	}

	if logs.Audit != nil {
		m["audit"] = strconv.FormatBool(aws.ToBool(logs.Audit))
	}

	return []interface{}{m}
}

func expandLogs(engineType string, l []interface{}) *types.Logs {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	m := l[0].(map[string]interface{})

	logs := &types.Logs{}

	if v, ok := m["general"]; ok {
		logs.General = aws.Bool(v.(bool))
	}

	// When the engine type is "RabbitMQ", the parameter audit cannot be set at all.
	if v, ok := m["audit"]; ok {
		if v, null, _ := nullable.Bool(v.(string)).Value(); !null {
			if !strings.EqualFold(engineType, string(types.EngineTypeRabbitmq)) {
				logs.Audit = aws.Bool(v)
			}
		}
	}

	return logs
}

func flattenLDAPServerMetadata(apiObject *types.LdapServerMetadataOutput, password string) []interface{} {
	if apiObject == nil {
		return nil
	}

	tfMap := map[string]interface{}{}

	if v := apiObject.Hosts; v != nil {
		tfMap["hosts"] = v
	}
	if v := apiObject.RoleBase; v != nil {
		tfMap["role_base"] = aws.ToString(v)
	}
	if v := apiObject.RoleName; v != nil {
		tfMap["role_name"] = aws.ToString(v)
	}
	if v := apiObject.RoleSearchMatching; v != nil {
		tfMap["role_search_matching"] = aws.ToString(v)
	}
	if v := apiObject.RoleSearchSubtree; v != nil {
		tfMap["role_search_subtree"] = aws.ToBool(v)
	}
	if password != "" {
		tfMap["service_account_password"] = password
	}
	if v := apiObject.ServiceAccountUsername; v != nil {
		tfMap["service_account_username"] = aws.ToString(v)
	}
	if v := apiObject.UserBase; v != nil {
		tfMap["user_base"] = aws.ToString(v)
	}
	if v := apiObject.UserRoleName; v != nil {
		tfMap["user_role_name"] = aws.ToString(v)
	}
	if v := apiObject.UserSearchMatching; v != nil {
		tfMap["user_search_matching"] = aws.ToString(v)
	}
	if v := apiObject.UserSearchSubtree; v != nil {
		tfMap["user_search_subtree"] = aws.ToBool(v)
	}

	return []interface{}{tfMap}
}

func expandLDAPServerMetadata(tfList []interface{}) *types.LdapServerMetadataInput {
	if len(tfList) == 0 || tfList[0] == nil {
		return nil
	}

	apiObject := &types.LdapServerMetadataInput{}

	tfMap := tfList[0].(map[string]interface{})

	if v, ok := tfMap["hosts"]; ok && len(v.([]interface{})) > 0 {
		apiObject.Hosts = flex.ExpandStringValueList(v.([]interface{}))
	}
	if v, ok := tfMap["role_base"].(string); ok && v != "" {
		apiObject.RoleBase = aws.String(v)
	}
	if v, ok := tfMap["role_name"].(string); ok && v != "" {
		apiObject.RoleName = aws.String(v)
	}
	if v, ok := tfMap["role_search_matching"].(string); ok && v != "" {
		apiObject.RoleSearchMatching = aws.String(v)
	}
	if v, ok := tfMap["role_search_subtree"].(bool); ok {
		apiObject.RoleSearchSubtree = aws.Bool(v)
	}
	if v, ok := tfMap["service_account_password"].(string); ok && v != "" {
		apiObject.ServiceAccountPassword = aws.String(v)
	}
	if v, ok := tfMap["service_account_username"].(string); ok && v != "" {
		apiObject.ServiceAccountUsername = aws.String(v)
	}
	if v, ok := tfMap["user_base"].(string); ok && v != "" {
		apiObject.UserBase = aws.String(v)
	}
	if v, ok := tfMap["user_role_name"].(string); ok && v != "" {
		apiObject.UserRoleName = aws.String(v)
	}
	if v, ok := tfMap["user_search_matching"].(string); ok && v != "" {
		apiObject.UserSearchMatching = aws.String(v)
	}
	if v, ok := tfMap["user_search_subtree"].(bool); ok {
		apiObject.UserSearchSubtree = aws.Bool(v)
	}

	return apiObject
}

var ValidateBrokerName = validation.All(
	validation.StringLenBetween(1, 50),
	validation.StringMatch(regexache.MustCompile(`^[0-9A-Za-z_-]+$`), ""),
)
