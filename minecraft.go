// Copyright (C) 2022 Andrew Ayer
// Copyright (C) 2025 Arusekk
//
// Permission is hereby granted, free of charge, to any person obtaining a
// copy of this software and associated documentation files (the "Software"),
// to deal in the Software without restriction, including without limitation
// the rights to use, copy, modify, merge, publish, distribute, sublicense,
// and/or sell copies of the Software, and to permit persons to whom the
// Software is furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included
// in all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL
// THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR
// OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE,
// ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
// OTHER DEALINGS IN THE SOFTWARE.
//
// Except as contained in this notice, the name(s) of the above copyright
// holders shall not be used in advertising or otherwise to promote the
// sale, use or other dealings in this Software without prior written
// authorization.
//
// Andrew Ayer: src.agwa.name/go-listener/tlsutil
// Arusekk: modified from hello.go

package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"

	"golang.org/x/text/encoding/unicode"

	pk "github.com/Tnze/go-mc/net/packet"
)

type peekedConn struct {
	net.Conn
	reader io.Reader
}
func (conn peekedConn) Read(p []byte) (int, error) { return conn.reader.Read(p) }

func peekMinecraftHelloFromConn(conn net.Conn) (string, net.Conn, error) {
	hello, reader, err := PeekMinecraftHello(conn)
	return hello, peekedConn{Conn: conn, reader: reader}, err
}

func PeekMinecraftHello(reader io.Reader) (string, io.Reader, error) {
	peekedBytes := new(bytes.Buffer)
	hello, err := readMinecraftHello(io.TeeReader(reader, peekedBytes))
	if err != nil {
		log.Printf("Error reading hello: %v", err)
		return "", io.MultiReader(peekedBytes, reader), err
	}
	return hello, io.MultiReader(peekedBytes, reader), nil
}

func readMinecraftMultiple(r io.Reader, fields ...pk.FieldDecoder) error {
	for i, v := range(fields) {
		_, err := v.ReadFrom(r)
		if err != nil {
			return fmt.Errorf("scanning multiple[%d] error: %w", i, err)
		}
	}
	return nil
}

func readMinecraftHello(reader io.Reader) (string, error) {
	var (
		Length, ID          pk.VarInt
	)
	// receive handshake packet
	log.Printf("Peeking MC client hello...")
	err := readMinecraftMultiple(reader, &Length, &ID)
	if err != nil {
		return "", err
	}
	log.Printf("Peeking MC client hello: found %v %v %v", Length, ID, err)
	if int32(Length) == 0xfe && int32(ID) == 0x7a {
		var (
			length2          pk.UnsignedByte
			length3, length4 pk.UnsignedShort
			proto            pk.UnsignedByte
			port             pk.Int
		)
		_, err := length2.ReadFrom(reader)
		if err != nil {
			return "", err
		}
		descb := make([]byte, length2 * 2)
		_, err = io.ReadFull(reader, descb)
		if err != nil {
			return "", err
		}
		decoder := unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM).NewDecoder()
		desc, err := decoder.Bytes(descb)
		log.Printf("Peeking MC client hello: legacy [%v]", string(desc))

		err = readMinecraftMultiple(reader, &length3, &proto, &length4)
		if err != nil {
			return "", err
		}

		hostnameb := make([]byte, length4 * 2)
		_, err = io.ReadFull(reader, hostnameb)
		if err != nil {
			return "", err
		}
		hostname, err := decoder.Bytes(hostnameb)
		if err != nil {
			return "", err
		}
		log.Printf("Peeking MC client hello: legacy [%v]", string(hostname))

		_, err = port.ReadFrom(reader)
		if err != nil {
			return "", err
		}
		log.Printf("Peeking MC client hello: v1.6 %v:%v [proto %v]", string(hostname), port, proto)

		return string(hostname), nil
	} else {
		var (
			Protocol            pk.VarInt        // ignored
			ServerAddress       pk.String
			ServerPort          pk.UnsignedShort // ignored
		)
		if int32(Length) < 4 {
			return "", fmt.Errorf("length too short")
		}
		p := pk.Packet{ ID: int32(ID), Data: make([]byte, Length - 2) }
		_, err = io.ReadFull(reader, p.Data)
		if err != nil {
			return "", err
		}
		err = p.Scan(&Protocol, &ServerAddress, &ServerPort)
		log.Printf("Peeking MC client hello: v1.7 %v:%v [proto %v]", ServerAddress, ServerPort, Protocol)
		return string(ServerAddress), err
	}
}
