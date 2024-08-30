package main

import (
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/EngoEngine/glm"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/rajveermalviya/go-webgpu/wgpu"
	wgpuext_glfw "github.com/rajveermalviya/go-webgpu/wgpuext/glfw"

	_ "embed"
)

var forceFallbackAdapter = os.Getenv("WGPU_FORCE_FALLBACK_ADAPTER") == "1"

func init() {
	runtime.LockOSThread()

	switch os.Getenv("WGPU_LOG_LEVEL") {
	case "OFF":
		wgpu.SetLogLevel(wgpu.LogLevel_Off)
	case "ERROR":
		wgpu.SetLogLevel(wgpu.LogLevel_Error)
	case "WARN":
		wgpu.SetLogLevel(wgpu.LogLevel_Warn)
	case "INFO":
		wgpu.SetLogLevel(wgpu.LogLevel_Info)
	case "DEBUG":
		wgpu.SetLogLevel(wgpu.LogLevel_Debug)
	case "TRACE":
		wgpu.SetLogLevel(wgpu.LogLevel_Trace)
	}
}

type Transform struct {
	// model      [16]float32
	view       [16]float32
	projection [16]float32
}
type Vertex struct {
	pos      [4]float32
	texCoord [2]float32
}

var VertexBufferLayout = wgpu.VertexBufferLayout{
	ArrayStride: uint64(unsafe.Sizeof(Vertex{})),
	StepMode:    wgpu.VertexStepMode_Vertex,
	Attributes: []wgpu.VertexAttribute{
		{
			Format:         wgpu.VertexFormat_Float32x4,
			Offset:         0,
			ShaderLocation: 0,
		},
		{
			Format:         wgpu.VertexFormat_Float32x2,
			Offset:         4 * 4,
			ShaderLocation: 1,
		},
	},
}

// var InstanceBufferLayout = wgpu.VertexBufferLayout{
// 	ArrayStride: uint64(unsafe.Sizeof([16]float32{})),
// 	StepMode:    wgpu.VertexStepMode_Instance,
// 	Attributes: []wgpu.VertexAttribute{
// 		{
// 			Format:         wgpu.VertexFormat_Float32x4,
// 			Offset:         0,
// 			ShaderLocation: 2,
// 		},
// 		{
// 			Format:         wgpu.VertexFormat_Float32x4,
// 			Offset:         4 * 4,
// 			ShaderLocation: 3,
// 		},
// 		{
// 			Format:         wgpu.VertexFormat_Float32x4,
// 			Offset:         8 * 4,
// 			ShaderLocation: 4,
// 		},
// 		{
// 			Format:         wgpu.VertexFormat_Float32x4,
// 			Offset:         12 * 4,
// 			ShaderLocation: 5,
// 		},
// 	},
// }

func vertex(pos1, pos2, pos3, tc1, tc2 float32) Vertex {
	return Vertex{
		pos:      [4]float32{pos1, pos2, pos3, 1},
		texCoord: [2]float32{tc1, tc2},
	}
}

var vertexData = [...]Vertex{
	// top (0, 0, 1)
	vertex(-1, -1, 1, 0, 0),
	vertex(1, -1, 1, 1, 0),
	vertex(1, 1, 1, 1, 1),
	vertex(-1, 1, 1, 0, 1),
	// bottom (0, 0, -1)
	vertex(-1, 1, -1, 1, 0),
	vertex(1, 1, -1, 0, 0),
	vertex(1, -1, -1, 0, 1),
	vertex(-1, -1, -1, 1, 1),
	// right (1, 0, 0)
	vertex(1, -1, -1, 0, 0),
	vertex(1, 1, -1, 1, 0),
	vertex(1, 1, 1, 1, 1),
	vertex(1, -1, 1, 0, 1),
	// left (-1, 0, 0)
	vertex(-1, -1, 1, 1, 0),
	vertex(-1, 1, 1, 0, 0),
	vertex(-1, 1, -1, 0, 1),
	vertex(-1, -1, -1, 1, 1),
	// front (0, 1, 0)
	vertex(1, 1, -1, 1, 0),
	vertex(-1, 1, -1, 0, 0),
	vertex(-1, 1, 1, 0, 1),
	vertex(1, 1, 1, 1, 1),
	// back (0, -1, 0)
	vertex(1, -1, 1, 0, 0),
	vertex(-1, -1, 1, 1, 0),
	vertex(-1, -1, -1, 1, 1),
	vertex(1, -1, -1, 0, 1),
}

var indexData = [...]uint16{
	0, 1, 2, 2, 3, 0, // top
	4, 5, 6, 6, 7, 4, // bottom
	8, 9, 10, 10, 11, 8, // right
	12, 13, 14, 14, 15, 12, // left
	16, 17, 18, 18, 19, 16, // front
	20, 21, 22, 22, 23, 20, // back
}

const texelsSize = 256

func createTexels() (texels [texelsSize * texelsSize]uint8) {
	for id := 0; id < (texelsSize * texelsSize); id++ {
		cx := 3.0*float32(id%texelsSize)/float32(texelsSize-1) - 2.0
		cy := 2.0*float32(id/texelsSize)/float32(texelsSize-1) - 1.0
		x, y, count := float32(cx), float32(cy), uint8(0)
		for count < 0xFF && x*x+y*y < 4.0 {
			oldX := x
			x = x*x - y*y + cx
			y = 2.0*oldX*y + cy
			count += 1
		}
		texels[id] = count
	}

	return texels
}

func generateMatrix(camera *Camera, aspectRatio float32) Transform {
	projection := glm.Perspective(math.Pi/4, aspectRatio, 1, 1000)

	forward := camera.Rotation.Rotate(&glm.Vec3{0, 0, 1})
	target := forward.Add(&camera.Position)
	up := camera.Rotation.Rotate(&glm.Vec3{0, 1, 0})
	view := glm.LookAtV(
		&target,
		&camera.Position,
		&up,
	)

	return Transform{view: view, projection: projection}
}

//go:embed shader.wgsl
var shader string

type State struct {
	surface      *wgpu.Surface
	swapChain    *wgpu.SwapChain
	depth        *wgpu.RenderPassDepthStencilAttachment
	depthTexture *wgpu.Texture
	depthView    *wgpu.TextureView
	device       *wgpu.Device
	queue        *wgpu.Queue
	config       *wgpu.SwapChainDescriptor
	vertexBuf    *wgpu.Buffer
	instanceBuf  *wgpu.Buffer
	indexBuf     *wgpu.Buffer
	uniformBuf   *wgpu.Buffer
	pipeline     *wgpu.RenderPipeline
	bindGroup    *wgpu.BindGroup
	camera       Camera
}

func createDepthAttachment(device *wgpu.Device, config *wgpu.SwapChainDescriptor) (*wgpu.Texture, error) {
	return device.CreateTexture(&wgpu.TextureDescriptor{
		Size: wgpu.Extent3D{
			Width:              config.Width,
			Height:             config.Height,
			DepthOrArrayLayers: 1,
		},
		MipLevelCount: 1,
		SampleCount:   1,
		Dimension:     wgpu.TextureDimension_2D,
		Format:        wgpu.TextureFormat_Depth32Float,
		Usage:         wgpu.TextureUsage_RenderAttachment,
	})
}

func (s *State) createRenderPassDepthAttachmentView() (*wgpu.RenderPassDepthStencilAttachment, error) {
	if s.depthTexture != nil {
		s.depthTexture.Release()
	}
	if s.depthView != nil {
		s.depthView.Release()
	}
	depth, err := createDepthAttachment(s.device, s.config)
	if err != nil {
		return nil, err
	}
	s.depthTexture = depth
	// defer depth.Release()
	depthView, err := depth.CreateView(nil)
	if err != nil {
		return nil, err
	}
	s.depthView = depthView
	// defer depthView.Release()
	return &wgpu.RenderPassDepthStencilAttachment{
		View:              depthView,
		DepthLoadOp:       wgpu.LoadOp_Clear,
		DepthStoreOp:      wgpu.StoreOp_Store,
		DepthClearValue:   1.0,
		StencilLoadOp:     wgpu.LoadOp_Clear,
		StencilStoreOp:    wgpu.StoreOp_Store,
		StencilClearValue: wgpu.LimitU32Undefined,
		StencilReadOnly:   false,
	}, nil
}

var model [][16]float32 = make([][16]float32, 1_000_000)

func InitState(window *glfw.Window) (s *State, err error) {
	defer func() {
		if err != nil {
			s.Destroy()
			s = nil
		}
	}()
	s = &State{}

	s.camera.Rotation = glm.QuatLookAtV(&glm.Vec3{}, &glm.Vec3{0, 0, -1}, &glm.Vec3{0, 1, 0})

	pos := glm.Vec3{0, 0, 0}
	for i := range model {
		model[i] = glm.Translate3D(pos[0], pos[1], pos[2])
		r := glm.Vec3{rand.Float32()*2 - 1, rand.Float32()*2 - 1, rand.Float32()*2 - 1}
		r = r.Mul(5)
		pos = pos.Add(&r)
	}

	instance := wgpu.CreateInstance(nil)
	defer instance.Release()

	s.surface = instance.CreateSurface(wgpuext_glfw.GetSurfaceDescriptor(window))

	adapter, err := instance.RequestAdapter(&wgpu.RequestAdapterOptions{
		ForceFallbackAdapter: forceFallbackAdapter,
		CompatibleSurface:    s.surface,
		PowerPreference:      wgpu.PowerPreference_HighPerformance,
	})
	if err != nil {
		return s, err
	}
	defer adapter.Release()

	s.device, err = adapter.RequestDevice(nil)
	if err != nil {
		return s, err
	}
	s.queue = s.device.GetQueue()

	caps := s.surface.GetCapabilities(adapter)

	width, height := window.GetSize()
	s.config = &wgpu.SwapChainDescriptor{
		Usage:       wgpu.TextureUsage_RenderAttachment,
		Format:      caps.Formats[0],
		Width:       uint32(width),
		Height:      uint32(height),
		PresentMode: wgpu.PresentMode_Immediate,
		AlphaMode:   caps.AlphaModes[0],
	}

	s.swapChain, err = s.device.CreateSwapChain(s.surface, s.config)
	if err != nil {
		return s, err
	}
	s.depth, err = s.createRenderPassDepthAttachmentView()
	s.vertexBuf, err = s.device.CreateBufferInit(&wgpu.BufferInitDescriptor{
		Label:    "Vertex Buffer",
		Contents: wgpu.ToBytes(vertexData[:]),
		Usage:    wgpu.BufferUsage_Vertex,
	})
	if err != nil {
		return s, err
	}

	s.indexBuf, err = s.device.CreateBufferInit(&wgpu.BufferInitDescriptor{
		Label:    "Index Buffer",
		Contents: wgpu.ToBytes(indexData[:]),
		Usage:    wgpu.BufferUsage_Index,
	})
	if err != nil {
		return s, err
	}

	{
		s.instanceBuf, err = s.device.CreateBuffer(&wgpu.BufferDescriptor{
			Size:             uint64(unsafe.Sizeof(model[0]) * uintptr(len(model))),
			Usage:            wgpu.BufferUsage_Storage | wgpu.BufferUsage_CopyDst,
			MappedAtCreation: false,
		})
		if err != nil {
			return s, err
		}
	}

	texels := createTexels()
	textureExtent := wgpu.Extent3D{
		Width:              texelsSize,
		Height:             texelsSize,
		DepthOrArrayLayers: 1,
	}
	texture, err := s.device.CreateTexture(&wgpu.TextureDescriptor{
		Size:          textureExtent,
		MipLevelCount: 1,
		SampleCount:   1,
		Dimension:     wgpu.TextureDimension_2D,
		Format:        wgpu.TextureFormat_R8Uint,
		Usage:         wgpu.TextureUsage_TextureBinding | wgpu.TextureUsage_CopyDst,
	})
	if err != nil {
		return s, err
	}
	defer texture.Release()

	textureView, err := texture.CreateView(nil)
	if err != nil {
		return s, err
	}
	defer textureView.Release()

	s.queue.WriteTexture(
		texture.AsImageCopy(),
		wgpu.ToBytes(texels[:]),
		&wgpu.TextureDataLayout{
			Offset:       0,
			BytesPerRow:  texelsSize,
			RowsPerImage: wgpu.CopyStrideUndefined,
		},
		&textureExtent,
	)

	{

		mxTotal := generateMatrix(&s.camera, float32(s.config.Width)/float32(s.config.Height))
		mxTotalBytes := *(*[unsafe.Sizeof(mxTotal)]byte)(unsafe.Pointer(&mxTotal))
		s.uniformBuf, err = s.device.CreateBufferInit(&wgpu.BufferInitDescriptor{
			Label:    "Uniform Buffer",
			Contents: mxTotalBytes[:],
			Usage:    wgpu.BufferUsage_Uniform | wgpu.BufferUsage_CopyDst,
		})
		if err != nil {
			return s, err
		}
	}

	shader, err := s.device.CreateShaderModule(&wgpu.ShaderModuleDescriptor{
		Label:          "shader.wgsl",
		WGSLDescriptor: &wgpu.ShaderModuleWGSLDescriptor{Code: shader},
	})
	if err != nil {
		return s, err
	}
	defer shader.Release()

	s.pipeline, err = s.device.CreateRenderPipeline(&wgpu.RenderPipelineDescriptor{
		Vertex: wgpu.VertexState{
			Module:     shader,
			EntryPoint: "vs_main",
			Buffers:    []wgpu.VertexBufferLayout{VertexBufferLayout},
		},
		Fragment: &wgpu.FragmentState{
			Module:     shader,
			EntryPoint: "fs_main",
			Targets: []wgpu.ColorTargetState{
				{
					Format:    s.config.Format,
					Blend:     nil,
					WriteMask: wgpu.ColorWriteMask_All,
				},
			},
		},
		Primitive: wgpu.PrimitiveState{
			Topology:  wgpu.PrimitiveTopology_TriangleList,
			FrontFace: wgpu.FrontFace_CCW,
			CullMode:  wgpu.CullMode_Back,
		},
		DepthStencil: &wgpu.DepthStencilState{
			Format:            wgpu.TextureFormat_Depth32Float,
			DepthWriteEnabled: true,
			DepthCompare:      wgpu.CompareFunction_Less,
			StencilFront: wgpu.StencilFaceState{
				Compare: wgpu.CompareFunction_Always,
			},
			StencilBack: wgpu.StencilFaceState{
				Compare: wgpu.CompareFunction_Always,
			},
		},
		Multisample: wgpu.MultisampleState{
			Count:                  1,
			Mask:                   0xFFFFFFFF,
			AlphaToCoverageEnabled: false,
		},
	})
	if err != nil {
		return s, err
	}

	bindGroupLayout := s.pipeline.GetBindGroupLayout(0)
	defer bindGroupLayout.Release()

	s.bindGroup, err = s.device.CreateBindGroup(&wgpu.BindGroupDescriptor{
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
				Buffer:  s.instanceBuf,
				Size:    wgpu.WholeSize,
			},
		},
	})
	if err != nil {
		return s, err
	}

	return s, nil
}

func (s *State) Resize(width, height int) {
	if width > 0 && height > 0 {
		s.config.Width = uint32(width)
		s.config.Height = uint32(height)

		// mxTotal := generateMatrix(&s.camera, float32(width)/float32(height))
		// s.queue.WriteBuffer(s.uniformBuf, 0, wgpu.ToBytes(mxTotal[:]))

		if s.swapChain != nil {
			s.swapChain.Release()
		}
		var err error
		s.swapChain, err = s.device.CreateSwapChain(s.surface, s.config)
		if err != nil {
			panic(err)
		}
		s.depth, err = s.createRenderPassDepthAttachmentView()
		if err != nil {
			panic(err)
		}

	}
}

var numThreads = runtime.NumCPU()
var staging *wgpu.Buffer = nil

func (s *State) Render() error {
	nextTexture, err := s.swapChain.GetCurrentTextureView()
	if err != nil {
		return err
	}
	defer nextTexture.Release()

	encoder, err := s.device.CreateCommandEncoder(nil)
	if err != nil {
		return err
	}
	defer encoder.Release()

	renderPass := encoder.BeginRenderPass(&wgpu.RenderPassDescriptor{
		ColorAttachments: []wgpu.RenderPassColorAttachment{
			{
				View:       nextTexture,
				LoadOp:     wgpu.LoadOp_Clear,
				StoreOp:    wgpu.StoreOp_Store,
				ClearValue: wgpu.Color{R: 0.1, G: 0.2, B: 0.3, A: 1.0},
			},
		},
		DepthStencilAttachment: s.depth,
	})
	defer renderPass.Release()

	{

		mxTotal := generateMatrix(&s.camera, float32(s.config.Width)/float32(s.config.Height))
		mxTotalBytes := *(*[unsafe.Sizeof(mxTotal)]byte)(unsafe.Pointer(&mxTotal))
		s.queue.WriteBuffer(s.uniformBuf, 0, mxTotalBytes[:])
	}
	{

		// modelBytes := unsafe.Slice((*byte)(unsafe.Pointer(&model[0])), unsafe.Sizeof(model[0])*uintptr(len(model)))
		if staging == nil {
			staging, err = s.device.CreateBuffer(&wgpu.BufferDescriptor{
				Size:             uint64(unsafe.Sizeof(model[0]) * uintptr(len(model))),
				Usage:            wgpu.BufferUsage_CopySrc | wgpu.BufferUsage_MapWrite,
				MappedAtCreation: false,
			})
			if err != nil {
				return err
			}
		}
		_len := uint(unsafe.Sizeof(model[0]) * uintptr(len(model)))
		{
			wg := sync.WaitGroup{}
			staging.MapAsync(wgpu.MapMode_Write, 0, uint64(_len), func(status wgpu.BufferMapAsyncStatus) {
				if status != wgpu.BufferMapAsyncStatus_Success {
					return
				}
			})
			s.device.Poll(true, nil)
			byteMap := staging.GetMappedRange(0, _len)
			modelMap := unsafe.Slice((*[16]float32)(unsafe.Pointer(&byteMap[0])), len(model))
			for a := range numThreads {
				wg.Add(1)
				go func() {
					start := a * len(model) / numThreads
					end := (a + 1) * len(model) / numThreads
					for i := start; i < end; i++ {
						modelMap[i] = model[i]
					}
					// for i := range model {
					// 	rotation := glm.HomogRotate3D(float32(dt)*0.1, &axis)
					// 	m := glm.Mat4(model[i])
					// 	model[i] = m.Mul4(&rotation)
					// }
					wg.Done()
				}()
			}
			wg.Wait()
			err = staging.Unmap()
			if err != nil {
				return err
			}
		}

		err = encoder.CopyBufferToBuffer(staging, 0, s.instanceBuf, 0, uint64(_len))
		if err != nil {
			return err
		}
		// staging.Release()
		// s.queue.WriteBuffer(s.instanceBuf, 0, modelBytes)
	}

	renderPass.SetPipeline(s.pipeline)
	renderPass.SetBindGroup(0, s.bindGroup, nil)
	renderPass.SetIndexBuffer(s.indexBuf, wgpu.IndexFormat_Uint16, 0, wgpu.WholeSize)
	renderPass.SetVertexBuffer(0, s.vertexBuf, 0, wgpu.WholeSize)
	renderPass.DrawIndexed(uint32(len(indexData)), uint32(len(model)), 0, 0, 0)
	renderPass.End()

	cmdBuffer, err := encoder.Finish(nil)
	if err != nil {
		return err
	}
	defer cmdBuffer.Release()

	s.queue.Submit(cmdBuffer)
	s.swapChain.Present()

	return nil
}

func (s *State) Destroy() {
	if s.bindGroup != nil {
		s.bindGroup.Release()
		s.bindGroup = nil
	}
	if s.pipeline != nil {
		s.pipeline.Release()
		s.pipeline = nil
	}
	if s.uniformBuf != nil {
		s.uniformBuf.Release()
		s.uniformBuf = nil
	}
	if s.indexBuf != nil {
		s.indexBuf.Release()
		s.indexBuf = nil
	}
	if s.vertexBuf != nil {
		s.vertexBuf.Release()
		s.vertexBuf = nil
	}
	if s.swapChain != nil {
		s.swapChain.Release()
		s.swapChain = nil
	}
	if s.config != nil {
		s.config = nil
	}
	if s.queue != nil {
		s.queue.Release()
		s.queue = nil
	}
	if s.device != nil {
		s.device.Release()
		s.device = nil
	}
	if s.surface != nil {
		s.surface.Release()
		s.surface = nil
	}
	if s.depthTexture != nil {
		s.depthTexture.Release()
	}
	if s.depthView != nil {
		s.depthView.Release()
	}
	if staging != nil {
		staging.Release()
	}

}

func main() {
	if err := glfw.Init(); err != nil {
		panic(err)
	}
	defer glfw.Terminate()

	glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
	window, err := glfw.CreateWindow(640, 480, "go-webgpu with glfw", nil, nil)
	if err != nil {
		panic(err)
	}
	defer window.Destroy()

	s, err := InitState(window)
	if err != nil {
		panic(err)
	}
	defer s.Destroy()

	mouseX, mouseY := float32(0), float32(0)
	window.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)
	window.SetCursorPosCallback(func(w *glfw.Window, xpos, ypos float64) {
		mouseX += (float32(xpos) - float32(s.config.Width)/2) / 10
		mouseY -= (float32(ypos) - float32(s.config.Height)/2) / 10
		// mouseX -= float32(xpos)
		// mouseY -= float32(ypos)
		// mouseX = float32(xpos) - float32(s.config.Width)/2
		// mouseY = float32(ypos) - float32(s.config.Height)/2
		// fmt.Println("mouseX:", mouseX, "mouseY:", mouseY)

		rotx := glm.QuatRotate(mouseY/100, &glm.Vec3{1, 0, 0})
		roty := glm.QuatRotate(-mouseX/100, &glm.Vec3{0, 1, 0})
		s.camera.Rotation = roty.Mul(&rotx)
		// s.camera.Rotation = rotx.Mul(&s.camera.Rotation)
		// s.camera.Rotation = roty.Mul(&s.camera.Rotation)
		// forward := s.camera.Rotation.Rotate(&glm.Vec3{0, 0, 1})
		// target := s.camera.Position.Add(&forward)
		// s.camera.Rotation = glm.AnglesToQuat(mouseY, mouseX, 0, glm.XYX)
		// s.camera.Rotation = s.camera.Rotation.Mul(&glm.QuatRotate(mouseY, &glm.Vec3{1, 0, 0}))
		// fmt.Println("camera.Rotation:", s.camera.Rotation)
	})

	keys := map[glfw.Key]bool{}
	window.SetKeyCallback(func(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
		// Print resource usage on pressing 'R'
		// if key == glfw.KeyR && (action == glfw.Press || action == glfw.Repeat) {
		// 	report := s.instance.GenerateReport()
		// 	buf, _ := json.MarshalIndent(report, "", "  ")
		// 	fmt.Print(string(buf))
		// }

		if action == glfw.Press || action == glfw.Repeat {
			keys[key] = true

			// fmt.Println("camera.Position:", s.camera.Position)
		}

		if action == glfw.Release {
			delete(keys, key)
		}

		// if key == glfw.KeyS && (action == glfw.Press || action == glfw.Repeat) {
		// 	// s.camera.Position = glm.Vec3{0, 0, -3}
		// 	// s.camera.Rotation = glm.Quat{W: 1}
		// 	s.camera.Position = s.camera.Position.Add(&glm.Vec3{0, 0, -0.1})
		// 	fmt.Println("camera.Position:", s.camera.Position)
		// }
		// if key == glfw.KeyW && (action == glfw.Press || action == glfw.Repeat) {
		// 	// s.camera.Position = glm.Vec3{0, 0, -3}
		// 	// s.camera.Rotation = glm.Quat{W: 1}
		// 	s.camera.Position = s.camera.Position.Add(&glm.Vec3{0, 0, 0.1})
		// 	fmt.Println("camera.Position:", s.camera.Position)
		// }
	})

	window.SetSizeCallback(func(w *glfw.Window, width, height int) {
		s.Resize(width, height)
	})

	avg := time.Duration(0)
	frames := 0
	dt := glfw.GetTime()
	timer := time.NewTicker(time.Second)

	go func() {
		for {
			<-timer.C
			{
				_avg := float32(avg) / float32(frames)
				fps := float32(time.Second) / _avg
				fmt.Println("FPS:", fps)
				frames = 0
				avg = 0
			}
		}
	}()

	for !window.ShouldClose() {
		frames++
		frame := time.Now()
		dt = glfw.GetTime() - dt
		glfw.PollEvents()

		move := glm.Vec3{0, 0, 0}
		if keys[glfw.KeyW] {
			move = move.Add(&glm.Vec3{0, 0, -0.1})
		}
		if keys[glfw.KeyS] {
			move = move.Add(&glm.Vec3{0, 0, 0.1})
		}
		if keys[glfw.KeyA] {
			move = move.Add(&glm.Vec3{-0.1, 0, 0})
		}
		if keys[glfw.KeyD] {
			move = move.Add(&glm.Vec3{0.1, 0, 0})
		}
		if keys[glfw.KeyQ] {
			move = move.Add(&glm.Vec3{0, -0.1, 0})
		}
		if keys[glfw.KeyE] {
			move = move.Add(&glm.Vec3{0, 0.1, 0})
		}
		move = move.Mul(float32(dt))
		move = s.camera.Rotation.Rotate(&move)
		s.camera.Position = s.camera.Position.Add(&move)

		// model := make([][16]float32, 1)
		axis := glm.Vec3{1, 1, 1}
		axis = glm.NormalizeVec3(axis)
		wg := sync.WaitGroup{}

		for a := range numThreads {
			wg.Add(1)
			go func() {
				start := a * len(model) / numThreads
				end := (a + 1) * len(model) / numThreads
				for i := start; i < end; i++ {
					rotation := glm.HomogRotate3D(0.1, &axis)
					m := glm.Mat4(model[i])
					model[i] = m.Mul4(&rotation)
				}
				// for i := range model {
				// 	rotation := glm.HomogRotate3D(float32(dt)*0.1, &axis)
				// 	m := glm.Mat4(model[i])
				// 	model[i] = m.Mul4(&rotation)
				// }
				wg.Done()
			}()
		}
		wg.Wait()

		// for i := range model {
		// 	rotation := glm.HomogRotate3D(float32(dt)*0.1, &axis)
		// 	m := glm.Mat4(model[i])
		// 	model[i] = m.Mul4(&rotation)
		// }

		err := s.Render()
		window.SetCursorPos(float64(s.config.Width)/2, float64(s.config.Height)/2)
		if err != nil {
			fmt.Println("error occured while rendering:", err)

			errstr := err.Error()
			switch {
			case strings.Contains(errstr, "Surface timed out"): // do nothing
			case strings.Contains(errstr, "Surface is outdated"): // do nothing
			case strings.Contains(errstr, "Surface was lost"): // do nothing
			default:
				panic(err)
			}
		}
		avg += time.Since(frame)

		// if frames == 1000 {
		// 	avg = avg / 1000.0
		// 	fps := float32(time.Second / avg)
		// 	fmt.Println("FPS:", fps)
		// 	frames = 0
		// 	avg = 0
		// }
	}
}
