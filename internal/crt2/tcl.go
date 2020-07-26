// Copyright 2020 The Sqlite Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package crt2 // import "modernc.org/sqlite/internal/crt2"

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"modernc.org/crt/v3"
)

// const (
// 	TCL_GLOBAL_ONLY = 1
// )
//
// var (
// 	fToken           uint64
// 	nameOfExecutable string
// 	objects          = map[uintptr]objectNfo{}
// 	objectsMu        sync.Mutex
// )
//
// func nextToken() uintptr { return uintptr(atomic.AddUint64(&fToken, 1)) }
//
// type objectNfo struct {
// 	memcheckerNfo
// 	value interface{}
// }
//
// func addObject(o interface{}) uintptr {
// 	var v objectNfo
// 	v.value = o
// 	v.object = o
// 	v.pc, v.file, v.line, v.ok = runtime.Caller(1)
// 	h := nextToken()
// 	objectsMu.Lock()
// 	objects[h] = v
// 	objectsMu.Unlock()
// 	return h
// }
//
// func getObject(h uintptr) interface{} {
// 	if h <= 0 {
// 		panic(todo(""))
// 	}
//
// 	objectsMu.Lock()
// 	v, ok := objects[h]
// 	if !ok {
// 		panic(todo("", h))
// 	}
//
// 	objectsMu.Unlock()
// 	return v.value
// }
//
// func removeObject(h uintptr) {
// 	if h <= 0 {
// 		panic(todo(""))
// 	}
//
// 	objectsMu.Lock()
// 	if _, ok := objects[h]; !ok {
// 		panic(todo(""))
// 	}
//
// 	delete(objects, h)
// 	objectsMu.Unlock()
// }

func todo(s string, args ...interface{}) string { //TODO-
	switch {
	case s == "":
		s = fmt.Sprintf(strings.Repeat("%v ", len(args)), args...)
	default:
		s = fmt.Sprintf(s, args...)
	}
	pc, fn, fl, _ := runtime.Caller(1)
	f := runtime.FuncForPC(pc)
	var fns string
	if f != nil {
		fns = f.Name()
		if x := strings.LastIndex(fns, "."); x > 0 {
			fns = fns[x+1:]
		}
	}
	r := fmt.Sprintf("%s:%d:%s: TODOTODO %s", fn, fl, fns, s) //TODOOK
	fmt.Fprintf(os.Stdout, "%s\n", r)
	os.Stdout.Sync()
	return r
}

func trc(s string, args ...interface{}) string { //TODO-
	switch {
	case s == "":
		s = fmt.Sprintf(strings.Repeat("%v ", len(args)), args...)
	default:
		s = fmt.Sprintf(s, args...)
	}
	_, fn, fl, _ := runtime.Caller(1)
	r := fmt.Sprintf("\n%s:%d: TRC %s", fn, fl, s)
	fmt.Fprintf(os.Stdout, "%s\n", r)
	os.Stdout.Sync()
	return r
}

// int sched_yield(void);
func Xsched_yield(tls *crt.TLS) int32 {
	panic(todo(""))
}

// int pthread_create(pthread_t *thread, const pthread_attr_t *attr, void *(*start_routine) (void *), void *arg);
func Xpthread_create(tls *crt.TLS, thread, attr, start_routine, arg uintptr) int32 {
	panic(todo(""))
}

// int pthread_detach(pthread_t thread);
func Xpthread_detach(tls *crt.TLS, thread crt.Size_t) int32 {
	panic(todo(""))
}

// int ferror(FILE *stream);
func Xferror(tls *crt.TLS, stream uintptr) int32 {
	panic(todo(""))
}

// // int ftruncate(int fd, off_t length);
// func Xftruncate(tls *crt.TLS, fd int32, length crt.Ssize_t) int32 {
// 	panic(todo(""))
// }

// int fstat(int fd, struct stat *statbuf);
func Xfstat(tls *crt.TLS, fd int32, statbuf uintptr) int32 {
	panic(todo(""))
}

// // int rename(const char *oldpath, const char *newpath);
// func Xrename(tls *crt.TLS, oldpath, newpath uintptr) int32 {
// 	panic(todo(""))
// }

// int pthread_mutex_lock(pthread_mutex_t *mutex);
func Xpthread_mutex_lock(tls *crt.TLS, mutex uintptr) int32 {
	panic(todo(""))
}

// int pthread_cond_signal(pthread_cond_t *cond);
func Xpthread_cond_signal(tls *crt.TLS, cond uintptr) int32 {
	panic(todo(""))
}

// int pthread_mutex_unlock(pthread_mutex_t *mutex);
func Xpthread_mutex_unlock(tls *crt.TLS, mutex uintptr) int32 {
	panic(todo(""))
}

// int pthread_mutex_init(pthread_mutex_t *restrict mutex, const pthread_mutexattr_t *restrict attr);
func Xpthread_mutex_init(tls *crt.TLS, mutex, attr uintptr) int32 {
	panic(todo(""))
}

// int pthread_cond_init(pthread_cond_t *restrict cond, const pthread_condattr_t *restrict attr);
func Xpthread_cond_init(tls *crt.TLS, cond, attr uintptr) int32 {
	panic(todo(""))
}

// int pthread_cond_wait(pthread_cond_t *restrict cond, pthread_mutex_t *restrict mutex);
func Xpthread_cond_wait(tls *crt.TLS, cond, mutex uintptr) int32 {
	panic(todo(""))
}

// int pthread_cond_destroy(pthread_cond_t *cond);
func Xpthread_cond_destroy(tls *crt.TLS, cond uintptr) int32 {
	panic(todo(""))
}

// // int stat(const char *pathname, struct stat *statbuf);
// func Xstat(tls *crt.TLS, pathname, statbuf uintptr) int32 {
// 	panic(todo(""))
// }
//
// // int lstat(const char *pathname, struct stat *statbuf);
// func Xlstat(tls *crt.TLS, pathname, statbuf uintptr) int32 {
// 	panic(todo(""))
// }
//
// // struct dirent *readdir(DIR *dirp);
// func Xreaddir(tls *crt.TLS, dirp uintptr) uintptr {
// 	panic(todo(""))
// }

// int pthread_mutex_destroy(pthread_mutex_t *mutex);
func Xpthread_mutex_destroy(tls *crt.TLS, mutex uintptr) int32 {
	panic(todo(""))
}

// // ============================================================================
//
// // void *malloc(size_t size);
// func Xmalloc(tls *crt.TLS, size crt.Size_t) uintptr {
// 	p := crt.Xmalloc(tls, size)
// 	if p != 0 {
// 		Memcheck.add(p, size)
// 	}
// 	return p
// }
//
// // void *calloc(size_t nmemb, size_t size);
// func Xcalloc(tls *crt.TLS, n, size crt.Size_t) uintptr {
// 	p := crt.Xcalloc(tls, n, size)
// 	if p != 0 {
// 		Memcheck.add(p, n*size)
// 	}
// 	return p
// }
//
// // void *realloc(void *ptr, size_t size);
// func Xrealloc(tls *crt.TLS, ptr uintptr, size crt.Size_t) uintptr {
// 	panic(todo(""))
// }
//
// // void free(void *ptr);
// func Xfree(tls *crt.TLS, ptr uintptr) {
// 	if ptr != 0 {
// 		Memcheck.remove(ptr)
// 	}
// 	crt.Xfree(tls, ptr)
// }
//
// // void abort(void);
// func Xabort(tls *crt.TLS) {
// 	Xexit(tls, 1)
// }
//
// // void exit(int status);
// func Xexit(tls *crt.TLS, status int32) {
// 	s := Memcheck.Audit()
// 	//trc("Memcheck.Audit(): %s", s)
// 	if s != "" && status == 0 {
// 		status = 1
// 	}
// 	fmt.Fprintln(os.Stderr, s)
// 	os.Stderr.Sync()
// 	crt.Xexit(tls, status)
// }
//
// func X__builtin_exit(tls *crt.TLS, status int32) { Xexit(tls, status) }
//
// // void __assert_fail(const char * assertion, const char * file, unsigned int line, const char * function);
// func X__assert_fail(tls *crt.TLS, assertion, file uintptr, line uint32, function uintptr) {
// 	fmt.Fprintf(os.Stderr, "assertion failure: %s:%d.%s: %s\n", crt.GoString(file), line, crt.GoString(function), crt.GoString(assertion))
// 	os.Stderr.Sync()
// 	Xexit(tls, 1)
// }
//
// // void __builtin_trap (void)
// func X__builtin_trap(tls *crt.TLS) {
// 	fmt.Fprintf(os.Stderr, "%s\ntrap\n", debug.Stack())
// 	os.Stderr.Sync()
// 	Xexit(tls, 1)
// }
//
// // void __builtin_unreachable (void)
// func X__builtin_unreachable(tls *crt.TLS) {
// 	fmt.Fprintf(os.Stderr, "%s\nunrechable\n", debug.Stack())
// 	os.Stderr.Sync()
// 	Xexit(tls, 1)
// }
//
// // ============================================================================
//
// // int Tcl_UnregisterChannel(Tcl_Interp *interp, Tcl_Channel chan);
// func XTcl_UnregisterChannel(tls *crt.TLS, interp, chan1 uintptr) int32 {
// 	panic(todo(""))
// }
//
// // void Tcl_Free(char *ptr);
// func XTcl_Free(tls *crt.TLS, ptr uintptr) {
// 	panic(todo(""))
// }
//
// // char * Tcl_Alloc(unsigned int size);
// func XTcl_Alloc(tls *crt.TLS, size uint32) uintptr {
// 	panic(todo(""))
// }
//
// // Tcl_Channel Tcl_CreateChannel(const Tcl_ChannelType *typePtr, const char *chanName, ClientData instanceData, int mask);
// func XTcl_CreateChannel(tls *crt.TLS, typePtr, chanName, instanceData uintptr, mask int32) uintptr {
// 	panic(todo(""))
// }
//
// // const char * Tcl_GetChannelName(Tcl_Channel chan);
// func XTcl_GetChannelName(tls *crt.TLS, chan1 uintptr) uintptr {
// 	panic(todo(""))
// }
//
// // char * Tcl_GetStringFromObj(Tcl_Obj *objPtr, int *lengthPtr);
// func XTcl_GetStringFromObj(tls *crt.TLS, objPtr, lengthPtr uintptr) uintptr {
// 	panic(todo(""))
// }
//
// // void Tcl_RegisterChannel(Tcl_Interp *interp, Tcl_Channel chan);
// func XTcl_RegisterChannel(tls *crt.TLS, interp, chan1 uintptr) {
// 	panic(todo(""))
// }
//
// // void Tcl_SetResult(Tcl_Interp *interp, char *result, Tcl_FreeProc *freeProc);
// func XTcl_SetResult(tls *crt.TLS, interp, result, freeProc uintptr) {
// 	panic(todo(""))
// }
//
// // void TclFreeObj(Tcl_Obj *objPtr);
// func XTclFreeObj(tls *crt.TLS, objPtr uintptr) {
// 	panic(todo(""))
// }
//
// // void Tcl_AppendResult(Tcl_Interp *interp, ...);
// func XTcl_AppendResult(tls *crt.TLS, interp, va uintptr) {
// 	panic(todo(""))
// }
//
// // void Tcl_BackgroundError(Tcl_Interp *interp);
// func XTcl_BackgroundError(tls *crt.TLS, interp uintptr) { panic(todo("")) }
//
// // Tcl_CreateInterp creates a new interpreter structure and returns a token for
// // it. The token is required in calls to most other Tcl procedures, such as
// // Tcl_CreateCommand, Tcl_Eval, and Tcl_DeleteInterp. The token returned by
// // Tcl_CreateInterp may only be passed to Tcl routines called from the same
// // thread as the original Tcl_CreateInterp call. It is not safe for multiple
// // threads to pass the same token to Tcl's routines. The new interpreter is
// // initialized with the built-in Tcl commands and with standard variables like
// // tcl_platform and env. To bind in additional commands, call
// // Tcl_CreateCommand, and to create additional variables, call Tcl_SetVar.
// //
// // Tcl_Interp * Tcl_CreateInterp(void);
// func XTcl_CreateInterp(tls *crt.TLS) uintptr {
// 	ctx := tee.NewContext("", false)
// 	if err := registerBuiltinCommands(ctx); err != nil {
// 		panic(todo("", err))
// 	}
//
// 	return addObject(ctx)
// }
//
// // Tcl_CreateObjCommand defines a new command in interp and associates it with
// // procedure proc such that whenever name is invoked as a Tcl command (e.g.,
// // via a call to Tcl_EvalObjEx) the Tcl interpreter will call proc to process
// // the command.
// //
// // Tcl_CreateObjCommand deletes any existing command name already associated
// // with the interpreter (however see below for an exception where the existing
// // command is not deleted). It returns a token that may be used to refer to the
// // command in subsequent calls to Tcl_GetCommandName. If name contains any ::
// // namespace qualifiers, then the command is added to the specified namespace;
// // otherwise the command is added to the global namespace. If
// // Tcl_CreateObjCommand is called for an interpreter that is in the process of
// // being deleted, then it does not create a new command and it returns NULL.
// // proc should have arguments and result that match the type Tcl_ObjCmdProc:
// //
// //	typedef int Tcl_ObjCmdProc(
// //	        ClientData clientData,
// //	        Tcl_Interp *interp,
// //	        int objc,
// //	        Tcl_Obj *const objv[]);
// //
// // When proc is invoked, the clientData and interp parameters will be copies of
// // the clientData and interp arguments given to Tcl_CreateObjCommand.
// // Typically, clientData points to an application-specific data structure that
// // describes what to do when the command procedure is invoked. Objc and objv
// // describe the arguments to the command, objc giving the number of argument
// // values (including the command name) and objv giving the values of the
// // arguments. The objv array will contain objc values, pointing to the argument
// // values. Unlike argv[argv] used in a string-based command procedure,
// // objv[objc] will not contain NULL.
// //
// // Additionally, when proc is invoked, it must not modify the contents of the
// // objv array by assigning new pointer values to any element of the array (for
// // example, objv[2] = NULL) because this will cause memory to be lost and the
// // runtime stack to be corrupted. The const in the declaration of objv will
// // cause ANSI-compliant compilers to report any such attempted assignment as an
// // error. However, it is acceptable to modify the internal representation of
// // any individual value argument. For instance, the user may call
// // Tcl_GetIntFromObj on objv[2] to obtain the integer representation of that
// // value; that call may change the type of the value that objv[2] points at,
// // but will not change where objv[2] points.
// //
// // proc must return an integer code that is either TCL_OK, TCL_ERROR,
// // TCL_RETURN, TCL_BREAK, or TCL_CONTINUE. See the Tcl overview man page for
// // details on what these codes mean. Most normal commands will only return
// // TCL_OK or TCL_ERROR. In addition, if proc needs to return a non-empty
// // result, it can call Tcl_SetObjResult to set the interpreter's result. In the
// // case of a TCL_OK return code this gives the result of the command, and in
// // the case of TCL_ERROR this gives an error message. Before invoking a command
// // procedure, Tcl_EvalObjEx sets interpreter's result to point to a value
// // representing an empty string, so simple commands can return an empty result
// // by doing nothing at all.
// //
// // The contents of the objv array belong to Tcl and are not guaranteed to
// // persist once proc returns: proc should not modify them. Call
// // Tcl_SetObjResult if you want to return something from the objv array.
// //
// // Ordinarily, Tcl_CreateObjCommand deletes any existing command name already
// // associated with the interpreter. However, if the existing command was
// // created by a previous call to Tcl_CreateCommand, Tcl_CreateObjCommand does
// // not delete the command but instead arranges for the Tcl interpreter to call
// // the Tcl_ObjCmdProc proc in the future. The old string-based Tcl_CmdProc
// // associated with the command is retained and its address can be obtained by
// // subsequent Tcl_GetCommandInfo calls. This is done for backwards
// // compatibility.
// //
// // DeleteProc will be invoked when (if) name is deleted. This can occur through
// // a call to Tcl_DeleteCommand, Tcl_DeleteCommandFromToken, or
// // Tcl_DeleteInterp, or by replacing name in another call to
// // Tcl_CreateObjCommand. DeleteProc is invoked before the command is deleted,
// // and gives the application an opportunity to release any structures
// // associated with the command. DeleteProc should have arguments and result
// // that match the type Tcl_CmdDeleteProc:
// //
// //	typedef void Tcl_CmdDeleteProc(
// //	        ClientData clientData);
// //
// // The clientData argument will be the same as the clientData argument passed
// // to Tcl_CreateObjCommand.
// //
// // Tcl_Command Tcl_CreateObjCommand(Tcl_Interp *interp, const char *cmdName, Tcl_ObjCmdProc *proc, ClientData clientData, Tcl_CmdDeleteProc *deleteProc);
// func XTcl_CreateObjCommand(tls *crt.TLS, interp, cmdName0, proc, clientData, deleteProc uintptr) uintptr {
// 	ctx := getObject(interp).(*tee.Context)
// 	if ctx.BeingDeleted {
// 		return 0
// 	}
//
// 	cmdName := crt.GoString(cmdName0)
// 	cmd, rc := ctx.Proc(
// 		tee.NewString(cmdName), tee.NewNoValue(), tee.NewNoValue(),
// 		func(ctx *tee.Context, values ...tee.Value) tee.ReturnCode {
// 			panic(todo(""))
// 		},
// 		func(cmd *tee.Command) error {
// 			panic(todo(""))
// 		},
// 		clientData,
// 	)
// 	if rc != tee.Ok {
// 		panic(todo(""))
// 	}
//
// 	return addObject(cmd)
// }
//
// // char * Tcl_DStringAppend(Tcl_DString *dsPtr, const char *bytes, int length);
// func XTcl_DStringAppend(tls *crt.TLS, dsPtr, bytes uintptr, length int32) uintptr { panic(todo("")) }
//
// // char * Tcl_DStringAppendElement(Tcl_DString *dsPtr, const char *element);
// func XTcl_DStringAppendElement(tls *crt.TLS, dsPtr, element uintptr) uintptr { panic(todo("")) }
//
// // void Tcl_DStringFree(Tcl_DString *dsPtr);
// func XTcl_DStringFree(tls *crt.TLS, dsPtr uintptr) { panic(todo("")) }
//
// // void Tcl_DStringInit(Tcl_DString *dsPtr);
// func XTcl_DStringInit(tls *crt.TLS, dsPtr uintptr) { panic(todo("")) }
//
// // int Tcl_DeleteCommand(Tcl_Interp *interp, const char *cmdName);
// func XTcl_DeleteCommand(tls *crt.TLS, interp, cmdName uintptr) int32 { panic(todo("")) }
//
// // Tcl_Obj * Tcl_DuplicateObj(Tcl_Obj *objPtr);
// func XTcl_DuplicateObj(tls *crt.TLS, objPtr uintptr) uintptr { panic(todo("")) }
//
// // int Tcl_Eval(Tcl_Interp *interp, const char *script);
// func XTcl_Eval(tls *crt.TLS, interp, script uintptr) int32 { panic(todo("")) }
//
// // int Tcl_EvalObjEx(Tcl_Interp *interp, Tcl_Obj *objPtr, int flags);
// func XTcl_EvalObjEx(tls *crt.TLS, interp, objPtr uintptr, flags int32) int32 { panic(todo("")) }
//
// // void Tcl_FindExecutable(const char *argv0);
// func XTcl_FindExecutable(tls *crt.TLS, argv00 uintptr) {
// 	argv0 := crt.GoString(argv00)
// 	if filepath.IsAbs(argv0) {
// 		nameOfExecutable = argv0
// 		return
// 	}
//
// 	argv0, err := exec.LookPath(argv0)
// 	if err != nil {
// 		return
// 	}
//
// 	if argv0, err = filepath.Abs(argv0); err != nil {
// 		return
// 	}
//
// 	nameOfExecutable = argv0
// }
//
// // int Tcl_GetBooleanFromObj(Tcl_Interp *interp, Tcl_Obj *objPtr, int *boolPtr);
// func XTcl_GetBooleanFromObj(tls *crt.TLS, interp, objPtr, boolPtr uintptr) int32 { panic(todo("")) }
//
// // unsigned char * Tcl_GetByteArrayFromObj(Tcl_Obj *objPtr, int *lengthPtr);
// func XTcl_GetByteArrayFromObj(tls *crt.TLS, objPtr, lengthPtr uintptr) uintptr { panic(todo("")) }
//
// // int Tcl_GetCharLength(Tcl_Obj *objPtr);
// func XTcl_GetCharLength(tls *crt.TLS, objPtr uintptr) int32 { panic(todo("")) }
//
// // int Tcl_GetDoubleFromObj(Tcl_Interp *interp, Tcl_Obj *objPtr, double *doublePtr);
// func XTcl_GetDoubleFromObj(tls *crt.TLS, interp, objPtr, doublePtr uintptr) int32 { panic(todo("")) }
//
// // int Tcl_GetIndexFromObjStruct(Tcl_Interp *interp, Tcl_Obj *objPtr, const void *tablePtr, int offset, const char *msg, int flags, int *indexPtr);
// func XTcl_GetIndexFromObjStruct(tls *crt.TLS, interp, objPtr, tablePtr uintptr, offset int32, msg uintptr, flags int32, indexPtr uintptr) int32 {
// 	panic(todo(""))
// }
//
// // int Tcl_GetIntFromObj(Tcl_Interp *interp, Tcl_Obj *objPtr, int *intPtr);
// func XTcl_GetIntFromObj(tls *crt.TLS, interp, objPtr, intPtr uintptr) int32 { panic(todo("")) }
//
// // Tcl_Obj * Tcl_GetObjResult(Tcl_Interp *interp);
// func XTcl_GetObjResult(tls *crt.TLS, interp uintptr) uintptr { panic(todo("")) }
//
// // char * Tcl_GetString(Tcl_Obj *objPtr);
// func XTcl_GetString(tls *crt.TLS, objPtr uintptr) uintptr { panic(todo("")) }
//
// // const char * Tcl_GetStringResult(Tcl_Interp *interp);
// func XTcl_GetStringResult(tls *crt.TLS, interp uintptr) uintptr { panic(todo("")) }
//
// // const char * Tcl_GetVar2(Tcl_Interp *interp, const char *part1, const char *part2, int flags);
// func XTcl_GetVar2(tls *crt.TLS, interp, part1, part2 uintptr, flags int32) uintptr { panic(todo("")) }
//
// // Tcl_Obj * Tcl_GetVar2Ex(Tcl_Interp *interp, const char *part1, const char *part2, int flags);
// func XTcl_GetVar2Ex(tls *crt.TLS, interp, part1, part2 uintptr, flags int32) uintptr { panic(todo("")) }
//
// // void Tcl_GetVersion(int *major, int *minor, int *patchLevel, int *type);
// func XTcl_GetVersion(tls *crt.TLS, major, minor, patchLevel, type1 uintptr) { panic(todo("")) }
//
// // int Tcl_GetWideIntFromObj(Tcl_Interp *interp, Tcl_Obj *objPtr, Tcl_WideInt *widePtr);
// func XTcl_GetWideIntFromObj(tls *crt.TLS, interp, objPtr, widePtr uintptr) int32 { panic(todo("")) }
//
// // int Tcl_GlobalEval(Tcl_Interp *interp, const char *command);
// func XTcl_GlobalEval(tls *crt.TLS, interp, command uintptr) int32 { panic(todo("")) }
//
// //  int Tcl_ListObjAppendElement(Tcl_Interp *interp, Tcl_Obj *listPtr, Tcl_Obj *objPtr);
// func XTcl_ListObjAppendElement(tls *crt.TLS, interp, listPtr, objPtr uintptr) int32 { panic(todo("")) }
//
// // int Tcl_ListObjGetElements(Tcl_Interp *interp, Tcl_Obj *listPtr, int *objcPtr, Tcl_Obj ***objvPtr);
// func XTcl_ListObjGetElements(tls *crt.TLS, interp, listPtr, objcPtr, objvPtr uintptr) int32 {
// 	panic(todo(""))
// }
//
// // int Tcl_ListObjIndex(Tcl_Interp *interp, Tcl_Obj *listPtr, int index, Tcl_Obj **objPtrPtr);
// func XTcl_ListObjIndex(tls *crt.TLS, interp, listPtr uintptr, index int32, objPtrPtr uintptr) int32 {
// 	panic(todo(""))
// }
//
// // int Tcl_ListObjLength(Tcl_Interp *interp, Tcl_Obj *listPtr, int *lengthPtr);
// func XTcl_ListObjLength(tls *crt.TLS, interp, listPtr, lengthPtr uintptr) int32 { panic(todo("")) }
//
// // void Tcl_NRAddCallback(Tcl_Interp *interp, Tcl_NRPostProc *postProcPtr, ClientData data0, ClientData data1, ClientData data2, ClientData data3);
// func XTcl_NRAddCallback(tls *crt.TLS, interp, postProcPtr, data0, data1, data2, data3 uintptr) {
// 	panic(todo(""))
// }
//
// // int Tcl_NRCallObjProc(Tcl_Interp *interp, Tcl_ObjCmdProc *objProc, ClientData clientData, int objc, Tcl_Obj *const objv[]);
// func XTcl_NRCallObjProc(tls *crt.TLS, interp, objProc, clientData uintptr, objc int32, objv uintptr) int32 {
// 	panic(todo(""))
// }
//
// // Tcl_Command Tcl_NRCreateCommand(Tcl_Interp *interp, const char *cmdName, Tcl_ObjCmdProc *proc, Tcl_ObjCmdProc *nreProc, ClientData clientData, Tcl_CmdDeleteProc *deleteProc);
// func XTcl_NRCreateCommand(tls *crt.TLS, interp, cmdName, proc, nreProc, clientData, deleteProc uintptr) uintptr {
// 	panic(todo(""))
// }
//
// // int Tcl_NREvalObj(Tcl_Interp *interp, Tcl_Obj *objPtr, int flags);
// func XTcl_NREvalObj(tls *crt.TLS, interp, objPtr uintptr, flags int32) int32 { panic(todo("")) }
//
// // Tcl_Obj * Tcl_NewByteArrayObj(const unsigned char *bytes, int length);
// func XTcl_NewByteArrayObj(tls *crt.TLS, bytes uintptr, length int32) uintptr { panic(todo("")) }
//
// // Tcl_Obj * Tcl_NewDoubleObj(double doubleValue);
// func XTcl_NewDoubleObj(tls *crt.TLS, doubleValue float64) uintptr { panic(todo("")) }
//
// // Tcl_Obj * Tcl_NewIntObj(int intValue);
// func XTcl_NewIntObj(tls *crt.TLS, intValue int32) uintptr { panic(todo("")) }
//
// // Tcl_Obj * Tcl_NewListObj(int objc, Tcl_Obj *const objv[]);
// func XTcl_NewListObj(tls *crt.TLS, objc int32, objv uintptr) uintptr { panic(todo("")) }
//
// // Tcl_Obj * Tcl_NewObj(void);
// func XTcl_NewObj(tls *crt.TLS) uintptr { panic(todo("")) }
//
// // Tcl_Obj * Tcl_NewStringObj(const char *bytes, int length);
// func XTcl_NewStringObj(tls *crt.TLS, bytes uintptr, length int32) uintptr { panic(todo("")) }
//
// // Tcl_Obj * Tcl_NewWideIntObj(Tcl_WideInt wideValue);
// func XTcl_NewWideIntObj(tls *crt.TLS, wideValue crt.Intptr) uintptr { panic(todo("")) }
//
// // Tcl_Obj * Tcl_ObjSetVar2(Tcl_Interp *interp, Tcl_Obj *part1Ptr, Tcl_Obj *part2Ptr, Tcl_Obj *newValuePtr, int flags);
// func XTcl_ObjSetVar2(tls *crt.TLS, interp, part1, part2, newValuePtr uintptr, flags int32) uintptr {
// 	panic(todo(""))
// }
//
// // int Tcl_PkgProvideEx(Tcl_Interp *interp, const char *name, const char *version, const void *clientData);
// func XTcl_PkgProvideEx(tls *crt.TLS, interp, name0, version0, clientData uintptr) int32 {
// 	name := crt.GoString(name0)
// 	version := crt.GoString(version0)
// 	switch name {
// 	case "sqlite3":
// 		if version != sqlite3.SQLITE_VERSION {
// 			panic(todo("%q %q", name, version))
// 		}
// 	default:
// 		panic(todo("%q %q", name, version))
// 	}
//
// 	return int32(tee.Ok)
// }
//
// // void Tcl_ResetResult(Tcl_Interp *interp);
// func XTcl_ResetResult(tls *crt.TLS, interp uintptr) { panic(todo("")) }
//
// // void Tcl_SetIntObj(Tcl_Obj *objPtr, int intValue);
// func XTcl_SetIntObj(tls *crt.TLS, objPtr uintptr, intValue int32) { panic(todo("")) }
//
// // void Tcl_SetObjResult(Tcl_Interp *interp, Tcl_Obj *resultObjPtr);
// func XTcl_SetObjResult(tls *crt.TLS, interp, resultObjPtr uintptr) { panic(todo("")) }
//
// // Tcl_SetSystemEncoding sets the default encoding that should be used whenever
// // the user passes a NULL value for the encoding argument to any of the other
// // encoding functions. If name is NULL, the system encoding is reset to the
// // default system encoding, binary. If the name did not refer to any known or
// // loadable encoding, TCL_ERROR is returned and an error message is left in
// // interp. Otherwise, this procedure increments the reference count of the new
// // system encoding, decrements the reference count of the old system encoding,
// // and returns TCL_OK.
// //
// // int Tcl_SetSystemEncoding(Tcl_Interp *interp, const char *name);
// func XTcl_SetSystemEncoding(tls *crt.TLS, interp, name0 uintptr) int32 {
// 	name := crt.GoString(name0)
// 	if interp != 0 {
// 		panic(todo(""))
// 	}
//
// 	if name != "utf-8" {
// 		panic(todo(""))
// 	}
//
// 	return int32(tee.Ok)
// }
//
// // Tcl_SetVar2Ex, Tcl_SetVar, Tcl_SetVar2, and Tcl_ObjSetVar2 will create a new
// // variable or modify an existing one. These procedures set the given variable
// // to the value given by newValuePtr or newValue and return a pointer to the
// // variable's new value, which is stored in Tcl's variable structure.
// // Tcl_SetVar2Ex and Tcl_ObjSetVar2 take the new value as a Tcl_Obj and return
// // a pointer to a Tcl_Obj. Tcl_SetVar and Tcl_SetVar2 take the new value as a
// // string and return a string; they are usually less efficient than
// // Tcl_ObjSetVar2. Note that the return value may be different than the
// // newValuePtr or newValue argument, due to modifications made by write traces.
// // If an error occurs in setting the variable (e.g. an array variable is
// // referenced without giving an index into the array) NULL is returned and an
// // error message is left in interp's result if the TCL_LEAVE_ERR_MSG flag bit
// // is set.
// //
// // The name of a variable may be specified to these procedures in four ways:
// //
// // 1. If Tcl_SetVar, Tcl_GetVar, or Tcl_UnsetVar is invoked, the variable name
// // is given as a single string, varName. If varName contains an open
// // parenthesis and ends with a close parenthesis, then the value between the
// // parentheses is treated as an index (which can have any string value) and the
// // characters before the first open parenthesis are treated as the name of an
// // array variable. If varName does not have parentheses as described above,
// // then the entire string is treated as the name of a scalar variable.
// //
// // 2. If the name1 and name2 arguments are provided and name2 is non-NULL, then
// // an array element is specified and the array name and index have already been
// // separated by the caller: name1 contains the name and name2 contains the
// // index. An error is generated if name1 contains an open parenthesis and ends
// // with a close parenthesis (array element) and name2 is non-NULL.
// //
// // 3. If name2 is NULL, name1 is treated just like varName in case [1] above
// // (it can be either a scalar or an array element variable name).
// //
// // The flags argument may be used to specify any of several options to the
// // procedures. It consists of an OR-ed combination of the following bits.
// //
// //	TCL_GLOBAL_ONLY
// //
// // Under normal circumstances the procedures look up variables as follows. If a
// // procedure call is active in interp, the variable is looked up at the current
// // level of procedure call. Otherwise, the variable is looked up first in the
// // current namespace, then in the global namespace. However, if this bit is set
// // in flags then the variable is looked up only in the global namespace even if
// // there is a procedure call active. If both TCL_GLOBAL_ONLY and
// // TCL_NAMESPACE_ONLY are given, TCL_GLOBAL_ONLY is ignored.
// //
// //	TCL_NAMESPACE_ONLY
// //
// // If this bit is set in flags then the variable is looked up only in the
// // current namespace; if a procedure is active its variables are ignored, and
// // the global namespace is also ignored unless it is the current namespace.
// //
// //	TCL_LEAVE_ERR_MSG
// //
// // If an error is returned and this bit is set in flags, then an error message
// // will be left in the interpreter's result, where it can be retrieved with
// // Tcl_GetObjResult or Tcl_GetStringResult. If this flag bit is not set then no
// // error message is left and the interpreter's result will not be modified.
// //
// //	TCL_APPEND_VALUE
// // If this bit is set then newValuePtr or newValue is appended to the current
// // value instead of replacing it. If the variable is currently undefined, then
// // the bit is ignored. This bit is only used by the Tcl_Set* procedures.
// //
// //	TCL_LIST_ELEMENT
// //
// // If this bit is set, then newValue is converted to a valid Tcl list element
// // before setting (or appending to) the variable. A separator space is appended
// // before the new list element unless the list element is going to be the first
// // element in a list or sublist (i.e. the variable's current value is empty, or
// // contains the single character “{”, or ends in “ }”). When appending, the
// // original value of the variable must also be a valid list, so that the
// // operation is the appending of a new list element onto a list.
// //
// // const char * Tcl_SetVar2(Tcl_Interp *interp, const char *part1, const char *part2, const char *newValue, int flags);
// func XTcl_SetVar2(tls *crt.TLS, interp, part10, part20, newValue0 uintptr, flags int32) {
// 	ctx := getObject(interp).(*tee.Context)
// 	part1 := crt.GoString(part10)
// 	part2 := crt.GoString(part20)
// 	newValue := crt.GoString(newValue0)
// 	if part20 != 0 && (strings.Contains(part1, "(") || strings.Contains(part1, ")")) {
// 		panic(todo(""))
// 	}
//
// 	name := part1
// 	if part2 != "" {
// 		name = fmt.Sprintf("%s(%s)", part1, part2)
// 	}
// 	switch flags {
// 	case TCL_GLOBAL_ONLY:
// 		_ = ctx
// 		panic(todo("%q %q %q %q %#x(%[5]v)", part1, part2, name, newValue, flags))
// 	default:
// 		panic(todo("%q %q %q %q %#x(%[5]v)", part1, part2, name, newValue, flags))
// 	}
// }
//
// // void Tcl_SetWideIntObj(Tcl_Obj *objPtr, Tcl_WideInt wideValue);
// func XTcl_SetWideIntObj(tls *crt.TLS, objPtr uintptr, wideValue crt.Intptr) { panic(todo("")) }
//
// // char * Tcl_TranslateFileName(Tcl_Interp *interp, const char *name, Tcl_DString *bufferPtr);
// func XTcl_TranslateFileName(tls *crt.TLS, interp, name, bufferPtr uintptr) uintptr { panic(todo("")) }
//
// // int Tcl_UnsetVar2(Tcl_Interp *interp, const char *part1, const char *part2, int flags);
// func XTcl_UnsetVar2(tls *crt.TLS, interp, part1, part2 uintptr, flags int32) { panic(todo("")) }
//
// // int Tcl_VarEval(Tcl_Interp *interp, ...);
// func XTcl_VarEval(tls *crt.TLS, interp, va uintptr) int32 { panic(todo("")) }
//
// // void Tcl_WrongNumArgs(Tcl_Interp *interp, int objc, Tcl_Obj *const objv[], const char *message);
// func XTcl_WrongNumArgs(tls *crt.TLS, interp uintptr, objc int32, objv, message uintptr) {
// 	panic(todo(""))
// }
//
// // int Tcl_GetCommandInfo(Tcl_Interp *interp, const char *cmdName, Tcl_CmdInfo *infoPtr);
// func XTcl_GetCommandInfo(tls *crt.TLS, interp, cmdName, infoPtr uintptr) int32 {
// 	panic(todo(""))
// }
//
// // Tcl_Command Tcl_CreateCommand(Tcl_Interp *interp, const char *cmdName, Tcl_CmdProc *proc, ClientData clientData, Tcl_CmdDeleteProc *deleteProc);
// func XTcl_CreateCommand(tls *crt.TLS, interp, cmdName, proc, clentData, deleteProc uintptr) uintptr {
// 	panic(todo(""))
// }
//
// // int Tcl_LinkVar(Tcl_Interp *interp, const char *varName, char *addr, int type);
// func XTcl_LinkVar(tls *crt.TLS, interp, varName, addr uintptr, type1 int32) int32 {
// 	panic(todo(""))
// }
//
// // void Tcl_AppendElement(Tcl_Interp *interp, const char *element);
// func XTcl_AppendElement(tls *crt.TLS, interp, element uintptr) {
// 	panic(todo(""))
// }
//
// // int Tcl_GetInt(Tcl_Interp *interp, const char *src, int *intPtr);
// func XTcl_GetInt(tls *crt.TLS, interp, src, intPtr uintptr) int32 {
// 	panic(todo(""))
// }
//
// // int Tcl_GetDouble(Tcl_Interp *interp, const char *src, double *doublePtr);
// func XTcl_GetDouble(tls *crt.TLS, interp, src, doublePtr uintptr) int32 {
// 	panic(todo(""))
// }
//
// // Tcl_Channel Tcl_GetChannel(Tcl_Interp *interp, const char *chanName, int *modePtr);
// func XTcl_GetChannel(tls *crt.TLS, interp, chanName, modePtr uintptr) uintptr {
// 	panic(todo(""))
// }
//
// // int Tcl_Flush(Tcl_Channel chan);
// func XTcl_Flush(tls *crt.TLS, chan1 uintptr) int32 {
// 	panic(todo(""))
// }
//
// // Tcl_WideInt Tcl_Seek(Tcl_Channel chan, Tcl_WideInt offset, int mode);
// func XTcl_Seek(tls *crt.TLS, chan1 uintptr, offset crt.Intptr, mode int32) crt.Intptr {
// 	panic(todo(""))
// }
//
// // ClientData Tcl_GetChannelInstanceData(Tcl_Channel chan);
// func XTcl_GetChannelInstanceData(tls *crt.TLS, chan1 uintptr) uintptr {
// 	panic(todo(""))
// }
//
// // int Tcl_EvalEx(Tcl_Interp *interp, const char *script, int numBytes, int flags);
// func XTcl_EvalEx(tls *crt.TLS, interp, script uintptr, numBytes, flags int32) int32 {
// 	panic(todo(""))
// }
//
// // __attribute__ ((noreturn)) void Tcl_Panic(const char *format, ...) __attribute__ ((__format__ (__printf__, 1, 2)));
// func XTcl_Panic(tls *crt.TLS, format, va uintptr) {
// 	panic(todo(""))
// }
//
// // int Tcl_GetBoolean(Tcl_Interp *interp, const char *src, int *boolPtr);
// func XTcl_GetBoolean(tls *crt.TLS, interp, src, boolPtr uintptr) int32 {
// 	panic(todo(""))
// }
//
// // char * Tcl_AttemptAlloc(unsigned int size);
// func XTcl_AttemptAlloc(tls *crt.TLS, size uint32) uintptr {
// 	panic(todo(""))
// }
//
// // char * Tcl_AttemptRealloc(char *ptr, unsigned int size);
// func XTcl_AttemptRealloc(tls *crt.TLS, ptr uintptr, size uint32) uintptr {
// 	panic(todo(""))
// }
//
// // int Tcl_UtfToLower(char *src);
// func XTcl_UtfToLower(tls *crt.TLS, src uintptr) int32 {
// 	panic(todo(""))
// }
//
// // void Tcl_AppendStringsToObj(Tcl_Obj *objPtr, ...);
// func XTcl_AppendStringsToObj(tls *crt.TLS, objPtr, va uintptr) {
// 	panic(todo(""))
// }
//
// // Tcl_HashEntry * Tcl_FirstHashEntry(Tcl_HashTable *tablePtr, Tcl_HashSearch *searchPtr);
// func XTcl_FirstHashEntry(tls *crt.TLS, tablePtr, searchPtr uintptr) uintptr {
// 	panic(todo(""))
// }
//
// // Tcl_HashEntry * Tcl_NextHashEntry(Tcl_HashSearch *searchPtr);
// func XTcl_NextHashEntry(tls *crt.TLS, searchPtr uintptr) uintptr {
// 	panic(todo(""))
// }
//
// // void Tcl_DeleteHashTable(Tcl_HashTable *tablePtr);
// func XTcl_DeleteHashTable(tls *crt.TLS, tablePtr uintptr) {
// 	panic(todo(""))
// }
//
// // void Tcl_InitHashTable(Tcl_HashTable *tablePtr, int keyType);
// func XTcl_InitHashTable(tls *crt.TLS, tablePtr uintptr, keyType int32) {
// 	panic(todo(""))
// }
//
// // int Tcl_SetCommandInfo(Tcl_Interp *interp, const char *cmdName, const Tcl_CmdInfo *infoPtr);
// func XTcl_SetCommandInfo(tls *crt.TLS, interp, cmdName, infoPtr uintptr) int32 {
// 	panic(todo(""))
// }
//
// // void Tcl_AppendObjToObj(Tcl_Obj *objPtr, Tcl_Obj *appendObjPtr);
// func XTcl_AppendObjToObj(tls *crt.TLS, objPtr, appendObjPtr uintptr) {
// 	panic(todo(""))
// }
//
// // Tcl_Obj * Tcl_ObjGetVar2(Tcl_Interp *interp, Tcl_Obj *part1Ptr, Tcl_Obj *part2Ptr, int flags);
// func XTcl_ObjGetVar2(tls *crt.TLS, interp, part1Ptr, part2Ptr uintptr, flags int32) uintptr {
// 	panic(todo(""))
// }
//
// // void Tcl_ThreadQueueEvent(Tcl_ThreadId threadId, Tcl_Event *evPtr, Tcl_QueuePosition position);
// func XTcl_ThreadQueueEvent(tls *crt.TLS, threadId, evPtr uintptr, position uint32) {
// 	panic(todo(""))
// }
//
// // void Tcl_ThreadAlert(Tcl_ThreadId threadId);
// func XTcl_ThreadAlert(tls *crt.TLS, threadId uintptr) {
// 	panic(todo(""))
// }
//
// // void Tcl_DeleteInterp(Tcl_Interp *interp);
// func XTcl_DeleteInterp(tls *crt.TLS, interp uintptr) {
// 	panic(todo(""))
// }
//
// // int Tcl_DoOneEvent(int flags);
// func XTcl_DoOneEvent(tls *crt.TLS, flags int32) int32 {
// 	panic(todo(""))
// }
//
// // void Tcl_ExitThread(int status);
// func XTcl_ExitThread(tls *crt.TLS, status int32) {
// 	panic(todo(""))
// }
//
// // Tcl_ThreadId Tcl_GetCurrentThread(void);
// func XTcl_GetCurrentThread(tls *crt.TLS) uintptr {
// 	panic(todo(""))
// }
//
// // int Tcl_CreateThread(Tcl_ThreadId *idPtr, Tcl_ThreadCreateProc *proc, ClientData clientData, int stackSize, int flags);
// func XTcl_CreateThread(tls *crt.TLS, idPtr, proc, clientData uintptr, stackSize, flags int32) int32 {
// 	panic(todo(""))
// }
//
// // void Tcl_GetTime(Tcl_Time *timeBuf);
// func XTcl_GetTime(tls *crt.TLS, timeBuf uintptr) {
// 	panic(todo(""))
// }
//
// // Tcl_Interp * Tcl_GetSlave(Tcl_Interp *interp, const char *slaveName);
// func XTcl_GetSlave(tls *crt.TLS, interp, slaveName uintptr) uintptr {
// 	panic(todo(""))
// }
//
// // ============================================================================
