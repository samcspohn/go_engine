package main

import (
	"fmt"
	"math"
	"os"
	"runtime"
	"shared"
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

var cubeVertexData = [...]Vertex{
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

var cubeIndexData = [...]uint16{
	0, 1, 2, 2, 3, 0, // top
	4, 5, 6, 6, 7, 4, // bottom
	8, 9, 10, 10, 11, 8, // right
	12, 13, 14, 14, 15, 12, // left
	16, 17, 18, 18, 19, 16, // front
	20, 21, 22, 22, 23, 20, // back
}
var planeVertexData = [...]Vertex{
	// top (0, 0, 1)
	vertex(-1, -1, 1, 0, 0),
	vertex(1, -1, 1, 1, 0),
	vertex(1, 1, 1, 1, 1),
	vertex(-1, 1, 1, 0, 1),
}

var planeIndexData = [...]uint16{
	0, 1, 2, 2, 3, 0, // top
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
	projection := glm.Perspective(math.Pi/4, aspectRatio, 0.1, 1_000)

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
	renderers    map[int]*Renderer
	uniformBuf   *wgpu.Buffer
	pipeline     *wgpu.RenderPipeline
	camera       Camera
	textureView  *wgpu.TextureView
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

var ECS = shared.NewECS()
var mu = sync.Mutex{}

func InitState(window *glfw.Window) (s *State, err error) {
	defer func() {
		if err != nil {
			s.Destroy()
			s = nil
		}
	}()
	s = &State{}

	s.camera.Rotation = glm.QuatLookAtV(&glm.Vec3{}, &glm.Vec3{0, 0, -1}, &glm.Vec3{0, 1, 0})

	// pos := glm.Vec3{0, 0, 0}
	// for i := range model {
	// 	axis := glm.Vec3{rand.Float32()*2 - 1, rand.Float32()*2 - 1, rand.Float32()*2 - 1}
	// 	axis = axis.Normalized()
	// 	rot := glm.HomogRotate3D(rand.Float32()*2*math.Pi, &axis)
	// 	m := glm.Mat4(model[i])
	// 	m = glm.Translate3D(pos[0], pos[1], pos[2])
	// 	m = m.Mul4(&rot)
	// 	model[i] = m
	// 	r := glm.Vec3{rand.Float32()*2 - 1, rand.Float32()*2 - 1, rand.Float32()*2 - 1}
	// 	r = r.Mul(5)
	// 	pos = pos.Add(&r)
	// }

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
	s.renderers = make(map[int]*Renderer)
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

	s.textureView, err = texture.CreateView(nil)
	if err != nil {
		return s, err
	}
	// defer textureView.Release()

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
	mu.Lock()
	for _, r := range s.renderers {
		r.Draw(s, encoder, renderPass)
	}
	mu.Unlock()
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
	// if s.bindGroup != nil {
	// 	s.bindGroup.Release()
	// 	s.bindGroup = nil
	// }
	if s.textureView != nil {
		s.textureView.Release()
		s.textureView = nil
	}
	if s.pipeline != nil {
		s.pipeline.Release()
		s.pipeline = nil
	}
	if s.uniformBuf != nil {
		s.uniformBuf.Release()
		s.uniformBuf = nil
	}
	if s.renderers != nil {
		for _, r := range s.renderers {
			r.Release()
		}
		s.renderers = nil
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

	s.renderers[0], err = createRenderer(s, "Cube", cubeVertexData[:], cubeIndexData[:], s.textureView)
	if err != nil {
		panic(err)
	}
	s.renderers[1], err = createRenderer(s, "Plane", cubeVertexData[:], cubeIndexData[:], s.textureView)
	if err != nil {
		panic(err)
	}
	s.renderers[2], err = createRenderer(s, "Bullets", cubeVertexData[:], cubeIndexData[:], s.textureView)
	if err != nil {
		panic(err)
	}
	plane := s.renderers[1]
	plane.numInstances = 1
	m := glm.Scale3D(40, 0.5, 40)
	t := glm.Translate3D(0, -4, 0)
	plane.instances[0] = t.Mul4(&m)
	s.renderers[1] = plane

	mouseX, mouseY := float32(0), float32(0)
	window.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)
	window.SetCursorPosCallback(func(w *glfw.Window, xpos, ypos float64) {
		mouseX += (float32(xpos) - float32(s.config.Width)/2) / 10
		mouseY -= (float32(ypos) - float32(s.config.Height)/2) / 10
		rotx := glm.QuatRotate(mouseY/100, &glm.Vec3{1, 0, 0})
		roty := glm.QuatRotate(-mouseX/100, &glm.Vec3{0, 1, 0})
		s.camera.Rotation = roty.Mul(&rotx)
	})

	keys := map[glfw.Key]bool{}
	window.SetKeyCallback(func(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
		if action == glfw.Press || action == glfw.Repeat {
			keys[key] = true
		}

		if action == glfw.Release {
			delete(keys, key)
		}
	})

	mouse := map[glfw.MouseButton]bool{}
	mouseup := map[glfw.MouseButton]bool{}
	mouseDown := map[glfw.MouseButton]bool{}
	window.SetMouseButtonCallback(func(w *glfw.Window, button glfw.MouseButton, action glfw.Action, mods glfw.ModifierKey) {
		if action == glfw.Press {
			mouse[button] = true
			mouseDown[button] = true
		}
		if action == glfw.Release {
			delete(mouse, button)
			mouseup[button] = true
		}
	})

	window.SetSizeCallback(func(w *glfw.Window, width, height int) {
		s.Resize(width, height)
	})

	avg := time.Duration(0)
	frames := 0
	// dt := glfw.GetTime()
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
	// Client()
	client := Client{}
	client.init()

	shared.Register(ECS, shared.Player{}, false)
	shared.Register(ECS, shared.Bullet{}, false)

	go client.Recv(func(str []byte) {
		offset := uintptr(0)
		mu.Lock()
		for offset < uintptr(len(str)) {
			t := (*shared.Submessage)(unsafe.Pointer(&str[offset]))
			i := uintptr(unsafe.Sizeof(shared.Submessage{}))
			if t.NumBytes > 0 {
				switch t.Op {
				case shared.OpUpdate:
					i = (*ECS.Entities[t.T]).SyncUpd(t, str[offset:])
				case shared.OpInstantiate:
					i = (*ECS.Entities[t.T]).SyncInst(t, str[offset:])
				case shared.OpDeinstantiate:
					i = (*ECS.Entities[t.T]).SyncDeinst(t, str[offset:])
				}
			}
			offset += i
		}
		mu.Unlock()
	})
	last_time := time.Now()
	for !window.ShouldClose() {
		frames++
		dt := time.Since(last_time).Seconds()
		last_time = time.Now()
		frame := time.Now()
		// dt = glfw.GetTime() - dt

		// println("dt:", dt)
		glfw.PollEvents()

		move := glm.Vec3{0, 0, 0}
		if keys[glfw.KeyW] {
			move = move.Add(&glm.Vec3{0, 0, -1})
		}
		if keys[glfw.KeyS] {
			move = move.Add(&glm.Vec3{0, 0, 1})
		}
		if keys[glfw.KeyA] {
			move = move.Add(&glm.Vec3{-1, 0, 0})
		}
		if keys[glfw.KeyD] {
			move = move.Add(&glm.Vec3{1, 0, 0})
		}
		if keys[glfw.KeyQ] {
			move = move.Add(&glm.Vec3{0, -1, 0})
		}
		if keys[glfw.KeyE] {
			move = move.Add(&glm.Vec3{0, 1, 0})
		}
		move = move.Mul(float32(dt) * 50.0)
		move = s.camera.Rotation.Rotate(&move)
		s.camera.Position = s.camera.Position.Add(&move)
		s.camera.Position[1] = float32(math.Max(float64(s.camera.Position.Y()), -3))
		// player := shared.Player{Position: s.camera.Position, Rotation: s.camera.Rotation, Id: int(client.id)}
		// message := (*[unsafe.Sizeof(player)]byte)(unsafe.Pointer(&player))
		// newMessage := append([]byte{0}, message[:]...)
		// client.Send(newMessage)
		playerMsg := shared.Player{Position: s.camera.Position, Rotation: s.camera.Rotation, Id: int(client.id)}
		playerUpd := []shared.Upd[shared.Player]{{Id: uint32(client.id), V: playerMsg}}
		msg := shared.EncodeSubmessage(playerUpd)

		if mouse[glfw.MouseButtonLeft] {
			// println("Mouse Down")
			// message := shared.Shoot{Position: s.camera.Position, Direction: s.camera.Rotation.Rotate(&glm.Vec3{0, 0, -1}), Speed: 50}
			dir := s.camera.Rotation.Rotate(&glm.Vec3{0, 0, -1})
			message := shared.Bullet{Position: s.camera.Position, Vel: dir.Mul(50)}
			bulletUpd := []shared.Inst[shared.Bullet]{{Id: 0, V: message}}
			msg = append(msg, shared.EncodeSubmessage(bulletUpd)...)
			// newMessage := append([]byte{1}, (*[unsafe.Sizeof(message)]byte)(unsafe.Pointer(&message))[:]...)

			// client.Send(newMessage)
		}
		client.Send(msg)

		mu.Lock()
		i := 0
		players := shared.GetStorage[shared.Player](ECS)
		for idx, player := range players.Data {
			if !players.Valid[idx] {
				continue
			}
			rotation := player.Rotation.Mat4()
			translation := glm.Translate3D(player.Position[0], player.Position[1], player.Position[2])
			m := translation.Mul4(&rotation)
			s.renderers[0].instances[i] = m
			i++
		}
		s.renderers[0].numInstances = i
		i = 0
		gravity := glm.Vec3{0, -9.8, 0}
		gravity = gravity.Mul(float32(dt))
		bullets := shared.GetStorage[shared.Bullet](ECS)
		for id, bullet := range bullets.Data {
			if i >= len(s.renderers[2].instances) {
				break
			}
			if !bullets.Valid[id] {
				continue
			}
			// simulation
			vel := bullet.Vel
			vel = vel.Mul(float32(dt))
			bullet.Position = bullet.Position.Add(&vel)
			bullet.Vel = bullet.Vel.Add(&gravity)
			bullets.Data[id] = bullet
			// matrix
			rotation := glm.QuatLookAtV(&glm.Vec3{}, &bullet.Vel, &glm.Vec3{0, 1, 0})
			rotMat := rotation.Mat4()
			translation := glm.Translate3D(bullet.Position[0], bullet.Position[1], bullet.Position[2])
			scale := glm.Scale3D(0.1, 0.1, 0.1)
			m := rotMat.Mul4(&scale)
			m = translation.Mul4(&m)
			s.renderers[2].instances[i] = m
			i++
		}
		s.renderers[2].numInstances = i
		mu.Unlock()
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

		mouseDown = map[glfw.MouseButton]bool{}
		mouseup = map[glfw.MouseButton]bool{}
	}
}
