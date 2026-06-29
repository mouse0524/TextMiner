#include <stdio.h>
#include "onnxruntime_c_api.h"
#ifdef _WIN32
#include <windows.h>
#include <locale.h>
#include <stdlib.h>
#endif

#define RETURN_ON_ERROR(expr) {      \
	OrtStatus* onnx_status = (expr); \
	if (onnx_status != NULL) {       \
		return onnx_status;          \
	}                                \
}

const OrtApi *MagikaGetApiBase() {
	const OrtApiBase* base = OrtGetApiBase();
	if (base == NULL) {
		fprintf(stderr, "OrtGetApiBase returned NULL\n");
		return NULL;
	}
	const OrtApi* api = base->GetApi(ORT_API_VERSION);
	if (api == NULL) {
		fprintf(stderr, "GetApi returned NULL\n");
		return NULL;
	}
	return api;
}

OrtStatus *MagikaCreateSession(const OrtApi *ort, const char *model, OrtSession **session, OrtMemoryInfo **memory_info) {
	//fprintf(stderr, "MagikaCreateSession called with model: %s\n", model);
	
	OrtEnv *env;
	RETURN_ON_ERROR(ort->CreateEnv(ORT_LOGGING_LEVEL_ERROR, "onnx", &env));
	//fprintf(stderr, "Created env\n");
	
	RETURN_ON_ERROR(ort->DisableTelemetryEvents(env));
	//fprintf(stderr, "Disabled telemetry\n");
	
	OrtSessionOptions *options;
	RETURN_ON_ERROR(ort->CreateSessionOptions(&options));
	//fprintf(stderr, "Created session options\n");
	
	RETURN_ON_ERROR(ort->EnableCpuMemArena(options));
	//fprintf(stderr, "Enabled CPU mem arena\n");
	
#ifdef _WIN32
	wchar_t model_path_w[MAX_PATH];
	MultiByteToWideChar(CP_UTF8, 0, model, -1, model_path_w, MAX_PATH);
	//fprintf(stderr, "Creating session with wide path\n");
	RETURN_ON_ERROR(ort->CreateSession(env, model_path_w, options, session));
#else
	RETURN_ON_ERROR(ort->CreateSession(env, model, options, session));
#endif
	//fprintf(stderr, "Created session\n");
	
	RETURN_ON_ERROR(ort->CreateCpuMemoryInfo(OrtArenaAllocator, OrtMemTypeDefault, memory_info));
	//fprintf(stderr, "Created CPU memory info\n");
	
	return NULL;
}

OrtStatus *MagikaRun(const OrtApi *ort, OrtSession *session, OrtMemoryInfo *memory_info, int32_t features[], int64_t sizeFeatures, float target[], int64_t sizeTarget) {
	const char *input_names[] = {"bytes"};
	const char *output_names[] = {"target_label"};
	const int64_t input_shape[] = {1, sizeFeatures};
	OrtValue *input_tensor = NULL;
	RETURN_ON_ERROR(ort->CreateTensorWithDataAsOrtValue(memory_info, features, sizeFeatures * sizeof(int32_t), input_shape, 2, ONNX_TENSOR_ELEMENT_DATA_TYPE_INT32, &input_tensor));
	OrtValue *output_tensor = NULL;
	RETURN_ON_ERROR(ort->Run(session, NULL, input_names, (const OrtValue *const *) &input_tensor, 1, output_names, 1, &output_tensor));
	float *out = NULL;
	RETURN_ON_ERROR(ort->GetTensorMutableData(output_tensor, (void **) &out));
	memcpy(target, out, sizeTarget * sizeof(float));
	ort->ReleaseValue(input_tensor);
	ort->ReleaseValue(output_tensor);
	return NULL;
}

const char *MagikaGetErrorMessage(const OrtStatus* onnx_status) {
	if (onnx_status == NULL) {
		return "";
	}
	return OrtGetApiBase()->GetApi(ORT_API_VERSION)->GetErrorMessage(onnx_status);
}
