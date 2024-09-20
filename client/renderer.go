package main

import (
	"unsafe"

	"github.com/rajveermalviya/go-webgpu/wgpu"
)

type Renderer struct {
	vertexBuf    *wgpu.Buffer
	vertexBufLen int
	indexBuf     *wgpu.Buffer
	indexBufLen  int
	instanceBuf  *wgpu.Buffer
	bindGroup    *wgpu.BindGroup
	instances    [][16]float32
	numInstances int
}

func (r *Renderer) Release() {
	if r.vertexBuf != nil {
		r.vertexBuf.Release()
	}
	if r.indexBuf != nil {
		r.indexBuf.Release()
	}
	if r.instanceBuf != nil {
		r.instanceBuf.Release()
	}
	if r.bindGroup != nil {
		r.bindGroup.Release()
	}

}
func createRenderer(s *State, name string, vertexData []Vertex, indexData []uint16, textureView *wgpu.TextureView) (*Renderer, error) {
	r := Renderer{}
	err := error(nil)
	r.vertexBuf, err = s.device.CreateBufferInit(&wgpu.BufferInitDescriptor{
		Label:    name + " Vertex Buffer",
		Contents: wgpu.ToBytes(vertexData[:]),
		Usage:    wgpu.BufferUsage_Vertex,
	})
	if err != nil {
		return &r, err
	}
	r.vertexBufLen = len(vertexData)

	r.indexBuf, err = s.device.CreateBufferInit(&wgpu.BufferInitDescriptor{
		Label:    name + " Index Buffer",
		Contents: wgpu.ToBytes(indexData[:]),
		Usage:    wgpu.BufferUsage_Index,
	})
	if err != nil {
		return &r, err
	}
	r.indexBufLen = len(indexData)

	r.instances = make([][16]float32, 10_000)

	{
		r.instanceBuf, err = s.device.CreateBuffer(&wgpu.BufferDescriptor{
			Size:             uint64(unsafe.Sizeof(r.instances[0]) * uintptr(len(r.instances))),
			Usage:            wgpu.BufferUsage_Storage | wgpu.BufferUsage_CopyDst,
			MappedAtCreation: false,
		})
		if err != nil {
			return &r, err
		}
	}
	bindGroupLayout := s.pipeline.GetBindGroupLayout(0)
	defer bindGroupLayout.Release()

	r.bindGroup, err = s.device.CreateBindGroup(&wgpu.BindGroupDescriptor{
		Layout: bindGroupLayout,
		Entries: []wgpu.BindGroupEntry{
			{
				Binding: 0,
				Buffer:  s.uniformBuf,
				Size:    wgpu.WholeSize,
			},
			{
				Binding:     1,
				TextureView: textureView,
				Size:        wgpu.WholeSize,
			},
			{
				Binding: 2,
				Buffer:  r.instanceBuf,
				Size:    wgpu.WholeSize,
			},
		},
	})
	return &r, err
}

func (r *Renderer) Draw(s *State, encoder *wgpu.CommandEncoder, renderPass *wgpu.RenderPassEncoder) error {

	// modelBytes := unsafe.Slice((*byte)(unsafe.Pointer(&model[0])), unsafe.Sizeof(model[0])*uintptr(len(model)))
	// if staging == nil {
	if r.numInstances == 0 {
		return nil
	}
	staging, err := s.device.CreateBuffer(&wgpu.BufferDescriptor{
		Size:             uint64(unsafe.Sizeof(r.instances[0]) * uintptr(r.numInstances)),
		Usage:            wgpu.BufferUsage_CopySrc | wgpu.BufferUsage_MapWrite,
		MappedAtCreation: true,
	})
	if err != nil {
		return err
	}
	// }
	_len := uint(unsafe.Sizeof(r.instances[0]) * uintptr(r.numInstances))
	{
		byteMap := staging.GetMappedRange(0, _len)
		modelMap := unsafe.Slice((*[16]float32)(unsafe.Pointer(&byteMap[0])), r.numInstances)
		copy(modelMap, r.instances)
		err = staging.Unmap()
		if err != nil {
			return err
		}
	}

	err = encoder.CopyBufferToBuffer(staging, 0, r.instanceBuf, 0, uint64(_len))
	if err != nil {
		return err
	}
	// staging.Release()
	// s.queue.WriteBuffer(s.instanceBuf, 0, modelBytes)

	renderPass.SetPipeline(s.pipeline)
	renderPass.SetBindGroup(0, r.bindGroup, nil)
	renderPass.SetIndexBuffer(r.indexBuf, wgpu.IndexFormat_Uint16, 0, wgpu.WholeSize)
	renderPass.SetVertexBuffer(0, r.vertexBuf, 0, wgpu.WholeSize)
	renderPass.DrawIndexed(uint32(r.indexBufLen), uint32(r.numInstances), 0, 0, 0)
	return nil
}
