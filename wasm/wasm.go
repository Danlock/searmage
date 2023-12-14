package wasm

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	_ "image/png"
	"log/slog"
	"os"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/emscripten"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"golang.org/x/exp/constraints"
)

//go:embed tesseract-core-simd-lstm.wasm
var tessWASM []byte

//go:embed eng.traineddata
var tessEngTrainedData []byte

//go:embed kubenav-logo.png
var testImg []byte

func printFuncDefs(funcs map[string]api.FunctionDefinition) string {
	funcDefs := ""
	for _, f := range funcs {
		funcDefs = fmt.Sprintf(funcDefs+"%s,", f.Name())
	}
	return funcDefs
}

func printFuncDefsArr(funcs []api.FunctionDefinition) string {
	funcDefs := ""
	for _, f := range funcs {
		funcDefs = fmt.Sprintf(funcDefs+"%s,", f.Name())
	}
	return funcDefs
}

func Run(ctx context.Context) error {
	waRT := wazero.NewRuntime(ctx)
	defer waRT.Close(ctx)

	compiledMod, err := waRT.CompileModule(ctx, tessWASM)
	if err != nil {
		return fmt.Errorf("wasm.Run waRT.CompileModule %w", err)
	}
	// slog.Log(ctx, slog.LevelInfo, "compiledMod",
	// 	"imports", printFuncDefsArr(compiledMod.ImportedFunctions()),
	// 	// "exports", printFuncDefs(compiledMod.ExportedFunctions()),
	// )
	err = defineImports(ctx, waRT, compiledMod)
	if err != nil {
		return fmt.Errorf("wasm.Run defineImports %w", err)
	}

	tessMod, err := waRT.InstantiateModule(ctx, compiledMod, wazero.NewModuleConfig().
		WithStderr(os.Stderr).
		WithStdout(os.Stdout).
		WithFSConfig(wazero.NewFSConfig().WithDirMount("/", "")).
		WithStartFunctions("_initialize"))
	if err != nil {
		return fmt.Errorf("wasm.Run waRT.InstantiateModule %w", err)
	}

	// describe module
	results, err := tessMod.ExportedFunction("TessBaseAPI_TessBaseAPI_0").Call(ctx)
	if err != nil || len(results) != 1 {
		return fmt.Errorf("wasm.Run TessBaseAPI_TessBaseAPI_0 results %v err %w", results, err)
	}
	tessBaseAPIPtr := results[0]

	engPtr, err := AllocateString(ctx, tessMod, "eng")
	if err != nil {
		return err
	}
	defer Free(ctx, tessMod, engPtr)

	engTrainDataPtr, err := AllocateBytes(ctx, tessMod, tessEngTrainedData)
	if err != nil {
		return err
	}
	defer Free(ctx, tessMod, engTrainDataPtr)

	results, err = tessMod.ExportedFunction("TessBaseAPI_Init_4").Call(ctx, tessBaseAPIPtr, engTrainDataPtr, uint64(len(tessEngTrainedData)), engPtr, 1 /* oem */)
	if err != nil || len(results) != 1 || api.DecodeI32(results[0]) == -1 {
		return fmt.Errorf("wasm.Run TessBaseAPI_Init_4 results %v err %w", results, err)
	}

	imgPtr, err := AllocateBytes(ctx, tessMod, testImg)
	if err != nil {
		return err
	}
	defer Free(ctx, tessMod, imgPtr)

	// char* EMSCRIPTEN_KEEPALIVE TessBaseAPI_ParseImage(TessBaseAPI* self, const Uint8Array image, int image_size, int exif, int angle) {

	results, err = tessMod.ExportedFunction("TessBaseAPI_ParseImage").Call(ctx, tessBaseAPIPtr, imgPtr, uint64(len(testImg)), 1, 0)
	if err != nil || len(results) != 1 {
		return fmt.Errorf("wasm.Run TessBaseAPI_ParseImage results %v err %w", results, err)
	}

	slog.Log(ctx, slog.LevelInfo, "TessBaseAPI_ParseImage",
		"results", results, "utf8", ReadString(tessMod.Memory(), results[0]),
	)
	return nil
}

func printFuncDef(fn api.Function) string {
	return fmt.Sprintf("%+v", fn.Definition())
}

func defineImports(ctx context.Context, waRT wazero.Runtime, compiledMod wazero.CompiledModule) error {
	wasi_snapshot_preview1.MustInstantiate(ctx, waRT)

	// wasiBuilder := waRT.NewHostModuleBuilder(wasi_snapshot_preview1.ModuleName)
	// wasi_snapshot_preview1.NewFunctionExporter().ExportFunctions(wasiBuilder)
	// wasiBuilder.NewFunctionBuilder().
	// 	WithFunc(func(_ context.Context, mod api.Module, fd, dirflags, path, pathLen, oflags int32, fs_rights_base, fs_rights_inheriting int64, fdflags_, opened_fd int32) {
	// 		pathBytes, _ := mod.Memory().Read(uint32(path), uint32(pathLen))
	// 		// _, err := pathOpen.Call(ctx, []uint64{
	// 		// 	uint64(fd), uint64(dirflags), uint64(path), uint64(pathLen), uint64(oflags), uint64(fs_rights_base), uint64(fs_rights_inheriting), uint64(fdflags_), uint64(opened_fd),
	// 		// }...)
	// 		slog.Log(ctx, slog.LevelInfo, "wasip1.path_open",
	// 			"path", string(pathBytes))
	// 	}).Export("path_open")
	// _, err := wasiBuilder.Instantiate(ctx)
	// if err != nil {
	// 	return err
	// }

	env := waRT.NewHostModuleBuilder("env")

	exporter, err := emscripten.NewFunctionExporterForModule(compiledMod)
	if err != nil {
		return fmt.Errorf("defineImports %w", err)
	}
	exporter.ExportFunctions(env)

	env.NewFunctionBuilder().WithFunc(func(ctx context.Context, mod api.Module, code, sigPtr, bufPtr int32) int32 {
		// emscripten_asm_const_int is used to run a few arbitrary JS calls that stuff a sort of progress counter into Module['TesseractProgress']
		// slog.Log(ctx, slog.LevelInfo, "env.emscripten_asm_const_int",
		// 	"code", code, "sigPtr", ReadString(mod.Memory(), sigPtr), "bufPtr", ReadString(mod.Memory(), bufPtr))
		return 0
	}).Export("emscripten_asm_const_int")

	env.NewFunctionBuilder().WithFunc(func(ctx context.Context, mod api.Module, commandPtr int32) int32 {
		// system seems to be used not only to check the operating system, but also to exec arbitrary commands.
		// Obviously that will not be happening here, but log the command string so we can see what the hell is being attempted.
		slog.Log(ctx, slog.LevelInfo, "env.system", "command", ReadString(mod.Memory(), commandPtr))
		// emscripten returns a 0 to indicate it's running in the browser without a shell. We shall do the same.
		return 0
	}).Export("system")

	env.NewFunctionBuilder().WithFunc(func(ctx context.Context, mod api.Module, name int32) {
		// tesseract.js embedded a pdf.tff file inside the wasm. If this file is actually needed we can just use go:embed instead.
		slog.Log(ctx, slog.LevelInfo, "env._emscripten_fs_load_embedded_files", "name", ReadString(mod.Memory(), name))
	}).Export("_emscripten_fs_load_embedded_files")

	env.NewFunctionBuilder().WithFunc(func(ctx context.Context, mod api.Module, buf, len int32) int32 {
		str, _ := mod.Memory().Read(uint32(buf), uint32(len))
		slog.Log(ctx, slog.LevelInfo, "env.__syscall_getcwd", "buf", buf, "len", len, "str", string(str))
		return -68
	}).Export("__syscall_getcwd")

	env.NewFunctionBuilder().WithFunc(func(ctx context.Context, mod api.Module, dirfd, path, flags int32) int32 {
		slog.Log(ctx, slog.LevelInfo, "env.__syscall_unlinkat", "dirfd", ReadString(mod.Memory(), dirfd), "path", ReadString(mod.Memory(), path), "flags", ReadString(mod.Memory(), flags))
		return 0
	}).Export("__syscall_unlinkat")

	env.NewFunctionBuilder().WithFunc(func(ctx context.Context, mod api.Module, path int32) int32 {
		slog.Log(ctx, slog.LevelInfo, "env.__syscall_rmdir", "path", ReadString(mod.Memory(), path))
		return 0
	}).Export("__syscall_rmdir")

	_, err = env.Instantiate(ctx)
	return err
}

// ReadString reads from the provided pointer until we reach a 0.
// If 0 is not found, returns an empty string.
func ReadString[T constraints.Integer](mem api.Memory, rawStrPtr T) string {
	strPtr := uint32(rawStrPtr)
	str, _ := mem.Read(strPtr, mem.Size()-strPtr)
	strEnd := bytes.IndexByte(str, 0)
	if strEnd == -1 {
		return ""
	}
	return string(str[:strEnd])
}

// AllocateString malloc's a string within the WASM modules memory. It also writes a 0 afterwards so it works
// for functions that only take in null terminated strings. Remember to call Free.
func AllocateString(ctx context.Context, mod api.Module, str string) (uint64, error) {
	results, err := mod.ExportedFunction("malloc").Call(ctx, uint64(len(str)+1))
	if err != nil || len(results) != 1 {
		return 0, fmt.Errorf("AllocateString _malloc results %v err %w", results, err)
	}
	strPtr := uint32(results[0])
	if !mod.Memory().WriteString(strPtr, str) {
		return 0, fmt.Errorf("AllocateString WriteString failed for %s", str)
	}
	if !mod.Memory().WriteByte(strPtr+uint32(len(str)), 0) {
		return 0, fmt.Errorf("AllocateString WriteByte 0 failed")
	}
	return results[0], nil
}

// AllocateBytes malloc's bytes within the WASM modules memory. Remember to call Free.
func AllocateBytes(ctx context.Context, mod api.Module, buf []byte) (uint64, error) {
	results, err := mod.ExportedFunction("malloc").Call(ctx, uint64(len(buf)+1))
	if err != nil || len(results) != 1 {
		return 0, fmt.Errorf("AllocateBytes _malloc results %v err %w", results, err)
	}
	ptr := uint32(results[0])
	if !mod.Memory().Write(ptr, buf) {
		return 0, fmt.Errorf("AllocateBytes Write failed for %d bytes", len(buf))
	}
	if !mod.Memory().WriteByte(ptr+uint32(len(buf)), 0) {
		return 0, fmt.Errorf("AllocateBytes WriteByte 0 failed")
	}
	return results[0], nil
}

func Free[T constraints.Integer](ctx context.Context, mod api.Module, ptr T) error {
	results, err := mod.ExportedFunction("free").Call(ctx, uint64(ptr))
	if err != nil || len(results) != 1 {
		return fmt.Errorf("wasm.Run _free results %v err %w", results, err)
	}
	return nil
}
