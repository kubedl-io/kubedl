package patch

import (
	"encoding/json"
	"strings"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	NullHolder    = "NULL_HOLDER"
	NullHolderStr = "\"NULL_HOLDER\""
)

type Patch struct {
	patchType types.PatchType
	patchData patchData
}

// Type implements Patch interface.
func (p *Patch) Type() types.PatchType {
	return p.patchType
}

// Data implements Patch interface.
func (p *Patch) Data(obj client.Object) ([]byte, error) {
	return []byte(p.String()), nil
}

func (p *Patch) String() string {
	js, _ := json.Marshal(&p.patchData)
	return strings.Replace(string(js), NullHolderStr, "null", -1)
}

// NewStrategicPatch returns a strategic-merge-patch type patch entity, which applies
// to build-in resource like pods, services...
func NewStrategicPatch() *Patch {
	return &Patch{patchType: types.StrategicMergePatchType}
}

// NewMergePatch returns a merge-patch type patch entity,  which applies to CRDs like
// tfjobs, pytorchjobs...
func NewMergePatch() *Patch {
	return &Patch{patchType: types.MergePatchType}
}

// Append/Remove finalizer only works when patch type is StrategicMergePatchType.

func (p *Patch) AddFinalizer(item string) *Patch {
	if p.patchType != types.StrategicMergePatchType {
		return p
	}

	if p.patchData.Meta == nil {
		p.patchData.Meta = &prunedMetadata{}
	}
	p.patchData.Meta.Finalizers = append(p.patchData.Meta.Finalizers, item)
	return p
}

func (p *Patch) RemoveFinalizer(item string) *Patch {
	if p.patchType != types.StrategicMergePatchType {
		return p
	}

	if p.patchData.Meta == nil {
		p.patchData.Meta = &prunedMetadata{}
	}
	p.patchData.Meta.DeletePrimitiveFinalizer = append(p.patchData.Meta.DeletePrimitiveFinalizer, item)
	return p
}

// Override finalizer only works when patch type is MergePatchType.

func (p *Patch) OverrideFinalizer(items []string) *Patch {
	if p.patchType != types.MergePatchType {
		return p
	}

	if p.patchData.Meta == nil {
		p.patchData.Meta = &prunedMetadata{}
	}
	p.patchData.Meta.Finalizers = items
	return p
}

func (p *Patch) InsertLabel(key, value string) *Patch {
	if p.patchData.Meta == nil {
		p.patchData.Meta = &prunedMetadata{}
	}
	if p.patchData.Meta.Labels == nil {
		p.patchData.Meta.Labels = make(map[string]string)
	}
	p.patchData.Meta.Labels[key] = value
	return p
}

func (p *Patch) DeleteLabel(key string) *Patch {
	if p.patchData.Meta == nil {
		p.patchData.Meta = &prunedMetadata{}
	}
	if p.patchData.Meta.Labels == nil {
		p.patchData.Meta.Labels = make(map[string]string)
	}
	p.patchData.Meta.Labels[key] = NullHolder
	return p
}

func (p *Patch) InsertAnnotation(key, value string) *Patch {
	if p.patchData.Meta == nil {
		p.patchData.Meta = &prunedMetadata{}
	}
	if p.patchData.Meta.Annotations == nil {
		p.patchData.Meta.Annotations = make(map[string]string)
	}
	p.patchData.Meta.Annotations[key] = value
	return p
}

func (p *Patch) DeleteAnnotation(key string) *Patch {
	if p.patchData.Meta == nil {
		p.patchData.Meta = &prunedMetadata{}
	}
	if p.patchData.Meta.Annotations == nil {
		p.patchData.Meta.Annotations = make(map[string]string)
	}
	p.patchData.Meta.Annotations[key] = NullHolder
	return p
}

type patchData struct {
	Meta *prunedMetadata `json:"metadata,omitempty"`
}

type prunedMetadata struct {
	Labels                   map[string]string `json:"labels,omitempty"`
	Annotations              map[string]string `json:"annotations,omitempty"`
	Finalizers               []string          `json:"finalizers,omitempty"`
	DeletePrimitiveFinalizer []string          `json:"$deleteFromPrimitiveList/finalizers,omitempty"`
}
