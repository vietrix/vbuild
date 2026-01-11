package config

func cloneOfflineSpec(spec *OfflineSpec) *OfflineSpec {
	if spec == nil {
		return nil
	}
	out := *spec
	if spec.Env != nil {
		out.Env = map[string]string{}
		for k, v := range spec.Env {
			out.Env[k] = v
		}
	}
	return &out
}

func mergeOfflineSpec(base, overlay *OfflineSpec) *OfflineSpec {
	if base == nil && overlay == nil {
		return nil
	}
	out := cloneOfflineSpec(base)
	if out == nil {
		out = &OfflineSpec{}
	}
	if overlay == nil {
		return out
	}
	if overlay.Enabled {
		out.Enabled = true
	}
	if overlay.Env != nil {
		out.Env = mergeStringMap(out.Env, overlay.Env)
	}
	return out
}

func cloneResourcePool(spec *ResourcePool) *ResourcePool {
	if spec == nil {
		return nil
	}
	out := *spec
	out.GPUDevices = append([]string(nil), spec.GPUDevices...)
	if spec.Groups != nil {
		out.Groups = map[string]int{}
		for k, v := range spec.Groups {
			out.Groups[k] = v
		}
	}
	return &out
}

func mergeResourcePool(base, overlay *ResourcePool) *ResourcePool {
	if base == nil && overlay == nil {
		return nil
	}
	out := cloneResourcePool(base)
	if out == nil {
		out = &ResourcePool{}
	}
	if overlay == nil {
		return out
	}
	if overlay.CPU != 0 {
		out.CPU = overlay.CPU
	}
	if overlay.Memory != "" {
		out.Memory = overlay.Memory
	}
	if overlay.GPUs != 0 {
		out.GPUs = overlay.GPUs
	}
	if len(overlay.GPUDevices) > 0 {
		out.GPUDevices = append([]string(nil), overlay.GPUDevices...)
	}
	if overlay.Groups != nil {
		if out.Groups == nil {
			out.Groups = map[string]int{}
		}
		for k, v := range overlay.Groups {
			out.Groups[k] = v
		}
	}
	return out
}

func mergeDatasetMap(base, overlay map[string]*Dataset) map[string]*Dataset {
	out := map[string]*Dataset{}
	for key, value := range base {
		out[key] = cloneDataset(value)
	}
	for key, value := range overlay {
		out[key] = cloneDataset(value)
	}
	return out
}

func cloneDataset(spec *Dataset) *Dataset {
	if spec == nil {
		return nil
	}
	out := *spec
	out.Files = append([]string(nil), spec.Files...)
	out.Tags = append([]string(nil), spec.Tags...)
	return &out
}

func mergeExperimentDefaults(base, overlay *ExperimentDefaults) *ExperimentDefaults {
	if base == nil && overlay == nil {
		return nil
	}
	out := &ExperimentDefaults{}
	if base != nil {
		*out = *base
		if base.Tags != nil {
			out.Tags = append([]string(nil), base.Tags...)
		}
		if base.Metadata != nil {
			out.Metadata = map[string]string{}
			for k, v := range base.Metadata {
				out.Metadata[k] = v
			}
		}
	}
	if overlay != nil {
		if overlay.Dir != "" {
			out.Dir = overlay.Dir
		}
		if overlay.Enabled {
			out.Enabled = true
		}
		if len(overlay.Tags) > 0 {
			out.Tags = append([]string(nil), overlay.Tags...)
		}
		if overlay.Metadata != nil {
			if out.Metadata == nil {
				out.Metadata = map[string]string{}
			}
			for k, v := range overlay.Metadata {
				out.Metadata[k] = v
			}
		}
	}
	return out
}

func cloneRegistrySpec(spec *RegistrySpec) *RegistrySpec {
	if spec == nil {
		return nil
	}
	out := *spec
	return &out
}

func cloneSnapshotSpec(spec *SnapshotSpec) *SnapshotSpec {
	if spec == nil {
		return nil
	}
	out := *spec
	return &out
}

func cloneSweepSpec(spec *SweepSpec) *SweepSpec {
	if spec == nil {
		return nil
	}
	out := *spec
	if spec.Grid != nil {
		out.Grid = map[string][]string{}
		for k, v := range spec.Grid {
			out.Grid[k] = append([]string(nil), v...)
		}
	}
	if spec.Sample != nil {
		out.Sample = map[string][]string{}
		for k, v := range spec.Sample {
			out.Sample[k] = append([]string(nil), v...)
		}
	}
	return &out
}

func cloneSplitSpec(spec *SplitSpec) *SplitSpec {
	if spec == nil {
		return nil
	}
	out := *spec
	return &out
}

func cloneValidateSpec(spec *ValidateSpec) *ValidateSpec {
	if spec == nil {
		return nil
	}
	out := *spec
	out.Paths = append([]string(nil), spec.Paths...)
	out.Extensions = append([]string(nil), spec.Extensions...)
	return &out
}

func cloneStatsSpec(spec *StatsSpec) *StatsSpec {
	if spec == nil {
		return nil
	}
	out := *spec
	out.Paths = append([]string(nil), spec.Paths...)
	return &out
}

func cloneMetricsSpec(spec *MetricsSpec) *MetricsSpec {
	if spec == nil {
		return nil
	}
	out := *spec
	out.Regex = append([]string(nil), spec.Regex...)
	return &out
}

func cloneCanarySpec(spec *CanarySpec) *CanarySpec {
	if spec == nil {
		return nil
	}
	out := *spec
	if spec.Rules != nil {
		out.Rules = map[string]CanaryRule{}
		for k, v := range spec.Rules {
			out.Rules[k] = v
		}
	}
	return &out
}

func cloneBenchmarkSpec(spec *BenchmarkSpec) *BenchmarkSpec {
	if spec == nil {
		return nil
	}
	out := *spec
	return &out
}

func cloneExperimentSpec(spec *ExperimentSpec) *ExperimentSpec {
	if spec == nil {
		return nil
	}
	out := *spec
	out.Tags = append([]string(nil), spec.Tags...)
	if spec.Metadata != nil {
		out.Metadata = map[string]string{}
		for k, v := range spec.Metadata {
			out.Metadata[k] = v
		}
	}
	return &out
}

func cloneSignSpec(spec *SignSpec) *SignSpec {
	if spec == nil {
		return nil
	}
	out := *spec
	return &out
}

func cloneSBOMSpec(spec *SBOMSpec) *SBOMSpec {
	if spec == nil {
		return nil
	}
	out := *spec
	return &out
}

func cloneCheckpointSpec(spec *CheckpointSpec) *CheckpointSpec {
	if spec == nil {
		return nil
	}
	out := *spec
	out.Paths = append([]string(nil), spec.Paths...)
	return &out
}

func cloneModelCardSpec(spec *ModelCardSpec) *ModelCardSpec {
	if spec == nil {
		return nil
	}
	out := *spec
	if spec.Metadata != nil {
		out.Metadata = map[string]string{}
		for k, v := range spec.Metadata {
			out.Metadata[k] = v
		}
	}
	return &out
}

func cloneNotebookSpec(spec *NotebookSpec) *NotebookSpec {
	if spec == nil {
		return nil
	}
	out := *spec
	if spec.Parameters != nil {
		out.Parameters = map[string]string{}
		for k, v := range spec.Parameters {
			out.Parameters[k] = v
		}
	}
	return &out
}

func cloneExportSpec(spec *ExportSpec) *ExportSpec {
	if spec == nil {
		return nil
	}
	out := *spec
	out.Include = append([]string(nil), spec.Include...)
	return &out
}
