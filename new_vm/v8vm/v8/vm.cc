#include "vm.h"
#include "v8.h"
#include "require.h"
#include "storage.h"
#include "snapshot_blob.bin.h"
#include "natives_blob.bin.h"

#include "libplatform/libplatform.h"

#include <assert.h>
#include <cstring>
#include <string>
#include <fstream>
#include <sstream>
#include <thread>
#include <stdlib.h>
#include <stdio.h>
#include <iostream>

using namespace v8;

#define NATIVE_LIB_PATH "v8/libjs/"

typedef struct {
  Persistent<Context> context;
  Isolate *isolate;
} Sandbox;

void init() {
    V8::InitializeICU();

    Platform *platform = platform::CreateDefaultPlatform();
    V8::InitializePlatform(platform);

    StartupData nativesData, snapshotData;
    nativesData.data = reinterpret_cast<char *>(natives_blob_bin);
    nativesData.raw_size = natives_blob_bin_len;
    snapshotData.data = reinterpret_cast<char *>(snapshot_blob_bin);
    snapshotData.raw_size = snapshot_blob_bin_len;
    V8::SetNativesDataBlob(&nativesData);
    V8::SetSnapshotDataBlob(&snapshotData);

    V8::Initialize();
    return;
}

IsolatePtr newIsolate() {
  Isolate::CreateParams create_params;
  create_params.array_buffer_allocator = ArrayBuffer::Allocator::NewDefaultAllocator();
  return static_cast<IsolatePtr>(Isolate::New(create_params));
}

void releaseIsolate(IsolatePtr ptr) {
    if (ptr == nullptr) {
        return;
    }

    Isolate *isolate = static_cast<Isolate*>(ptr);
    isolate->Dispose();
    return;
}

const char *copyString(const std::string &str) {
    char *cstr = new char[str.length() + 1];
    std::strcpy(cstr, str.c_str());
    return cstr;
}

std::string v8ValueToStdString(Local<Value> val) {
    String::Utf8Value str(val);
    if (str.length() == 0) {
        return "";
    }
    return *str;
}

//void nativeRequire(const FunctionCallbackInfo<Value> &info) {
//    Isolate *isolate = info.GetIsolate();
//
//    Local<Value> path = info[0];
//    if (!path->IsString()) {
//        Local<Value> err = Exception::Error(
//            String::NewFromUtf8(isolate, "_native_require empty path")
//        );
//        isolate->ThrowException(err);
//    }
//
//    String::Utf8Value pathStr(path);
//    std::string fullRelPath = std::string(NATIVE_LIB_PATH) + *pathStr;
//
//    std::ifstream f(fullRelPath);
//    std::stringstream buffer;
//    buffer << f.rdbuf();
//
//    Local<String> source = String::NewFromUtf8(isolate, buffer.str().c_str(), NewStringType::kNormal).ToLocalChecked();
//    Local<String> fileName = String::NewFromUtf8(isolate, *pathStr, NewStringType::kNormal).ToLocalChecked();
//    Local<Script> script = Script::Compile(source, fileName);
//
//    if (!script.IsEmpty()) {
//        Local<Value> result = script->Run();
//        if (!result.IsEmpty()) {
//            info.GetReturnValue().Set(result);
//        }
//    }
//}

void nativeLog(const FunctionCallbackInfo<Value> &info) {
    Isolate *isolate = info.GetIsolate();

    Local<Value> msg = info[0];
    if (!msg->IsString()) {
        Local<Value> err = Exception::Error(
            String::NewFromUtf8(isolate, "_native_log empty log")
        );
        isolate->ThrowException(err);
    }

    String::Utf8Value msgStr(msg);
    std::cout << "native_log: " << *msgStr << std::endl;
    return;
}

void nativeReadFile(const FunctionCallbackInfo<Value> &info) {
    Isolate *isolate = info.GetIsolate();

    Local<Value> fileName = info[0];
    if (!fileName->IsString()) {
        Local<Value> err = Exception::Error(
            String::NewFromUtf8(isolate, "_native_readFile empty file name.")
        );
        isolate->ThrowException(err);
    }

    String::Utf8Value fileNameStr(fileName);
    std::string fullRelPath = std::string(NATIVE_LIB_PATH) + *fileNameStr + std::string(".js");

    std::ifstream f(fullRelPath);
    std::stringstream buffer;
    buffer << f.rdbuf();

    info.GetReturnValue().Set(String::NewFromUtf8(isolate, buffer.str().c_str()));

    return;
}

void nativeRun(const FunctionCallbackInfo<Value> &info) {
    Isolate *isolate = info.GetIsolate();

    Local<Value> source = info[0];
    Local<Value> fileName = info[1];
    if (!fileName->IsString()) {
        Local<Value> err = Exception::Error(
            String::NewFromUtf8(isolate, "_native_run empty script.")
        );
        isolate->ThrowException(err);
    }

    Local<String> source2 = String::NewFromUtf8(isolate, v8ValueToStdString(source).c_str(), NewStringType::kNormal).ToLocalChecked();
    Local<String> fileName2 = String::NewFromUtf8(isolate, v8ValueToStdString(fileName).c_str(), NewStringType::kNormal).ToLocalChecked();
    Local<Script> script = Script::Compile(source2, fileName2);

    if (!script.IsEmpty()) {
        Local<Value> result = script->Run();
        if (!result.IsEmpty()) {
            info.GetReturnValue().Set(result);
        }
    }

    return;
}

Local<ObjectTemplate> createGlobalTpl(Isolate *isolate) {
    Local<ObjectTemplate> global = ObjectTemplate::New(isolate);
    global->SetInternalFieldCount(1);

    InitRequire(isolate, global);
    InitStorage(isolate, global);

    global->Set(
              String::NewFromUtf8(isolate, "_native_log", NewStringType::kNormal)
                  .ToLocalChecked(),
              v8::FunctionTemplate::New(isolate, nativeLog));

    global->Set(
                  String::NewFromUtf8(isolate, "_native_readFile", NewStringType::kNormal)
                      .ToLocalChecked(),
                  v8::FunctionTemplate::New(isolate, nativeReadFile));

    global->Set(
                      String::NewFromUtf8(isolate, "_native_run", NewStringType::kNormal)
                          .ToLocalChecked(),
                      v8::FunctionTemplate::New(isolate, nativeRun));

    return global;
}

SandboxPtr newSandbox(IsolatePtr ptr) {
    Isolate *isolate = static_cast<Isolate*>(ptr);
    Locker locker(isolate);

    Isolate::Scope isolate_scope(isolate);
    HandleScope handle_scope(isolate);

    Local<ObjectTemplate> globalTpl = createGlobalTpl(isolate);
    Local<Context> context = Context::New(isolate, NULL, globalTpl);
    Local<Object> global = context->Global();

    Sandbox *sbx = new Sandbox;
    global->SetInternalField(0, External::New(isolate, sbx));

    //sbx->context.Reset(isolate, Context::New(isolate, nullptr, globalTpl));
    sbx->context.Reset(isolate, context);
    sbx->isolate = isolate;

    return static_cast<SandboxPtr>(sbx);
}

void releaseSandbox(SandboxPtr ptr) {
    if (ptr == nullptr) {
        return;
    }

    Sandbox *sbx = static_cast<Sandbox*>(ptr);

    Locker locker(sbx->isolate);
    Isolate::Scope isolate_scope(sbx->isolate);

    sbx->context.Reset();
    return;
}

std::string report_exception(Isolate *isolate, Local<Context> ctx, TryCatch& tryCatch) {
    std::stringstream ss;
    ss << "Uncaught exception: ";

    ss << v8ValueToStdString(tryCatch.Exception());

    if (!tryCatch.Message().IsEmpty()) {
        if (!tryCatch.Message()->GetScriptResourceName()->IsUndefined()) {
            ss << std::endl;
            ss << "at " << v8ValueToStdString(tryCatch.Message()->GetScriptResourceName());

            Maybe<int> lineNo = tryCatch.Message()->GetLineNumber(ctx);
            Maybe<int> start = tryCatch.Message()->GetStartColumn(ctx);
            Maybe<int> end = tryCatch.Message()->GetEndColumn(ctx);
            MaybeLocal<String> sourceLine = tryCatch.Message()->GetSourceLine(ctx);

            if (lineNo.IsJust()) {
                ss << ":" << lineNo.ToChecked();
            }
            if (start.IsJust()) {
                ss << ":" << start.ToChecked();
            }
            if (!sourceLine.IsEmpty()) {
                ss << std::endl;
                ss << "  " << v8ValueToStdString(sourceLine.ToLocalChecked());
            }
            if (start.IsJust() && end.IsJust()) {
                ss << std::endl;
                ss << "  ";
                for (int i = 0; i < start.ToChecked(); i++) {
                    ss << " ";
                }
                for (int i = start.ToChecked(); i < end.ToChecked(); i++) {
                    ss << "^";
                }
            }
        }
    }

    if (!tryCatch.StackTrace().IsEmpty()) {
        ss << std::endl;
        ss << "Stack tree: " << std::endl;
        ss << v8ValueToStdString(tryCatch.StackTrace());
    }

    return ss.str();
}

const char* ToCString(const v8::String::Utf8Value& value) {
  return *value ? *value : "<string conversion failed>";
}

void LoadVM(Isolate *isolate) {
    std::string nativeModulePath = NATIVE_LIB_PATH "nativeModule.js";
    std::ifstream nf(nativeModulePath);
    std::stringstream nbuffer;
    nbuffer << nf.rdbuf();

    Local<String> source = String::NewFromUtf8(isolate, nbuffer.str().c_str(), NewStringType::kNormal).ToLocalChecked();
    Local<String> fileName = String::NewFromUtf8(isolate, nativeModulePath.c_str(), NewStringType::kNormal).ToLocalChecked();
    Local<Script> script = Script::Compile(source, fileName);

    if (!script.IsEmpty()) {
        Local<Value> result = script->Run();
        if (!result.IsEmpty()) {
//            std::cout << "result vm: " << v8ValueToStdString(result) << std::endl;
        }
    }

    std::string storagePath = NATIVE_LIB_PATH "storage.js";
    std::ifstream sf(storagePath);
    std::stringstream sbuffer;
    sbuffer << sf.rdbuf();

    source = String::NewFromUtf8(isolate, sbuffer.str().c_str(), NewStringType::kNormal).ToLocalChecked();
    fileName = String::NewFromUtf8(isolate, storagePath.c_str(), NewStringType::kNormal).ToLocalChecked();
    script = Script::Compile(source, fileName);

    if (!script.IsEmpty()) {
        Local<Value> result = script->Run();
        if (!result.IsEmpty()) {
//            std::cout << "result vm: " << v8ValueToStdString(result) << std::endl;
        }
    }
}

ValueTuple Execute(SandboxPtr ptr, const char *code) {
    Sandbox *sbx = static_cast<Sandbox*>(ptr);
    Isolate *isolate = sbx->isolate;

    Locker locker(isolate);
    Isolate::Scope isolate_scope(isolate);

    HandleScope handle_scope(isolate);
    Context::Scope context_scope(sbx->context.Get(isolate));

    LoadVM(isolate);

    TryCatch tryCatch(isolate);
    tryCatch.SetVerbose(false);

    Local<String> source = String::NewFromUtf8(isolate, code, NewStringType::kNormal).ToLocalChecked();
    Local<String> fileName = String::NewFromUtf8(isolate, "_default_name.js", NewStringType::kNormal).ToLocalChecked();
    Local<Script> script = Script::Compile(source, fileName);

    ValueTuple res = { nullptr, nullptr };
    if (script.IsEmpty()) {
        std::string exception = report_exception(isolate, sbx->context.Get(isolate), tryCatch);
        res.Err = copyString(exception);
        return res;
    }

    Local<Value> result = script->Run();

    if (result.IsEmpty()) {
        std::string exception = report_exception(isolate, sbx->context.Get(isolate), tryCatch);
        res.Err = copyString(exception);
    } else {
        String::Utf8Value retV8Str(isolate, result);
        res.Value = strdup(ToCString(retV8Str));
    }

    return res;
}
