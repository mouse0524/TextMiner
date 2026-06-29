//go:build cgo && onnxruntime && windows

package onnx

// #include "onnx_runtime.h"
import "C"

import (
	"fmt"
	"unsafe"
)

// NewOnnx returns an onnx that can perform inferences using an ONNX Runtime
// (https://onnxruntime.ai/) and the given model.
// It wraps the C calls to the ONNX Runtime API https://onnxruntime.ai/docs/api/c.
func NewOnnx(modelPath string, sizeTarget int) (Onnx, error) {
	api := C.MagikaGetApiBase()
	if api == nil {
		return nil, fmt.Errorf("get api base: nil api - ONNX Runtime DLL may not be loaded")
	}

	ort := &onnxRuntime{
		api:        api,
		sizeTarget: sizeTarget,
	}

	modelPathC := C.CString(modelPath)
	defer C.free(unsafe.Pointer(modelPathC))

	err := C.MagikaCreateSession(ort.api, modelPathC, &ort.session, &ort.memory)
	if err != nil {
		return nil, fmt.Errorf("create session: %v", C.GoString(C.MagikaGetErrorMessage(err)))
	}

	if ort.session == nil {
		return nil, fmt.Errorf("create session: nil session - model path: %s", modelPath)
	}

	if ort.memory == nil {
		return nil, fmt.Errorf("create memory info: nil memory info")
	}

	return ort, nil
}

// onnxRuntime implements the Onnx interface relying on a cgo call
// to a C ONNX Runtime library.
type onnxRuntime struct {
	api        *C.OrtApi
	session    *C.OrtSession
	memory     *C.OrtMemoryInfo
	sizeTarget int
}

func (ort *onnxRuntime) Run(features []int32) ([]float32, error) {
	if ort == nil {
		return nil, fmt.Errorf("run: nil onnxRuntime")
	}
	if ort.api == nil {
		return nil, fmt.Errorf("run: nil api")
	}
	if ort.session == nil {
		return nil, fmt.Errorf("run: nil session")
	}
	if ort.memory == nil {
		return nil, fmt.Errorf("run: nil memory")
	}

	target := make([]float32, ort.sizeTarget)
	if err := C.MagikaRun(ort.api, ort.session, ort.memory, (*C.int32_t)(&features[0]), C.int64_t(len(features)), (*C.float)(&target[0]), C.int64_t(len(target))); err != nil {
		return nil, fmt.Errorf("run: %v", C.GoString(C.MagikaGetErrorMessage(err)))
	}
	return target, nil
}
