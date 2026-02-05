package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/verda-cloud/verdacloud-sdk-go/pkg/verda"
)

func cpuObjectValue(cpu verda.InstanceCPU) (types.Object, diag.Diagnostics) {
	return types.ObjectValue(
		map[string]attr.Type{
			"description":     types.StringType,
			"number_of_cores": types.Int64Type,
		},
		map[string]attr.Value{
			"description":     types.StringValue(cpu.Description),
			"number_of_cores": types.Int64Value(int64(cpu.NumberOfCores)),
		},
	)
}

func gpuObjectValue(gpu verda.InstanceGPU) (types.Object, diag.Diagnostics) {
	return types.ObjectValue(
		map[string]attr.Type{
			"description":    types.StringType,
			"number_of_gpus": types.Int64Type,
		},
		map[string]attr.Value{
			"description":    types.StringValue(gpu.Description),
			"number_of_gpus": types.Int64Value(int64(gpu.NumberOfGPUs)),
		},
	)
}

func memoryObjectValue(memory verda.InstanceMemory) (types.Object, diag.Diagnostics) {
	return types.ObjectValue(
		map[string]attr.Type{
			"description":       types.StringType,
			"size_in_gigabytes": types.Int64Type,
		},
		map[string]attr.Value{
			"description":       types.StringValue(memory.Description),
			"size_in_gigabytes": types.Int64Value(int64(memory.SizeInGigabytes)),
		},
	)
}

func storageObjectValue(storage verda.InstanceStorage) (types.Object, diag.Diagnostics) {
	return types.ObjectValue(
		map[string]attr.Type{
			"description": types.StringType,
		},
		map[string]attr.Value{
			"description": types.StringValue(storage.Description),
		},
	)
}

func stringListValue(ctx context.Context, values []string) (types.List, diag.Diagnostics) {
	return types.ListValueFrom(ctx, types.StringType, values)
}

func objectListValue(attrTypes map[string]attr.Type, items []map[string]attr.Value) (types.List, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	var values []attr.Value

	for _, item := range items {
		obj, diags := types.ObjectValue(attrTypes, item)
		diagnostics.Append(diags...)
		values = append(values, obj)
	}

	listValue, diags := types.ListValue(types.ObjectType{AttrTypes: attrTypes}, values)
	diagnostics.Append(diags...)
	return listValue, diagnostics
}

func stringPointerValueOrNull(value *string) attr.Value {
	if value == nil || *value == "" {
		return types.StringNull()
	}
	return types.StringValue(*value)
}

func boolPointerValueOrNull(value *bool) attr.Value {
	if value == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*value)
}
