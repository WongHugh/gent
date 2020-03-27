//Time    : 2020-03-27 15:11
//Author  : Hugh
//File    : codec.go
//Descripe:

package codec

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/panjf2000/gnet"
)

var (

	// ErrUnexpectedEOF occurs when no enough data to read by codec.
	ErrUnexpectedEOF = errors.New("there is no enough data")

	// ErrUnsupportedLength occurs when unsupported lengthFieldLength is from input data.
	ErrUnsupportedLength = errors.New("unsupported lengthFieldLength. (expected: 1, 2, 3, 4, or 8)")
	// ErrTooLessLength occurs when adjusted frame length is less than zero.
	ErrTooLessLength = errors.New("adjusted frame length is less than zero")
)

// ICodec is the interface of gnet codec.
type ICodec interface {
	// Encode encodes frames upon server responses into TCP stream.
	Encode(c gnet.Conn, buf []byte) ([]byte, error)
	// Decode decodes frames from TCP stream via specific implementation.
	Decode(c gnet.Conn) ([]byte, error)
}

type LengthFieldBasedFrameCodec struct {
	encoderConfig EncoderConfig
	decoderConfig DecoderConfig
}



func NewLengthFieldBasedFrameCodec(encoderConfig EncoderConfig, decoderConfig DecoderConfig) *LengthFieldBasedFrameCodec {
	return &LengthFieldBasedFrameCodec{encoderConfig, decoderConfig}
}

// EncoderConfig config for encoder.
type EncoderConfig struct {
	// ByteOrder is the ByteOrder of the length field.
	ByteOrder binary.ByteOrder
	// LengthFieldLength is the length of the length field.
	LengthFieldLength int
	// LengthAdjustment is the compensation value to add to the value of the length field
	LengthAdjustment int
	// LengthIncludesLengthFieldLength is true, the length of the prepended length field is added to the value of the prepended length field
	LengthIncludesLengthFieldLength bool
	// header of the userData
	Header []byte
	//whether to add the crc check
	AddCheckData bool
}

// DecoderConfig config for decoder.
type DecoderConfig struct {
	// ByteOrder is the ByteOrder of the length field.
	ByteOrder binary.ByteOrder
	// LengthFieldOffset is the offset of the length field
	LengthFieldOffset int
	// LengthFieldLength is the length of the length field
	LengthFieldLength int
	// LengthAdjustment is the compensation value to add to the value of the length field
	LengthAdjustment int
	// InitialBytesToStrip is the number of first bytes to strip out from the decoded frame
	InitialBytesToStrip int
	//FinalBytesToStrip is the number of first bytes to strip out from the decoded frame
	FinalBytesToStrip int
	//whether to check the data
	CheckData bool
}

// Encode ...' buf 为用户数据
func (cc *LengthFieldBasedFrameCodec) Encode(c gnet.Conn, buf []byte) ([]byte, error) {
	length := len(buf) + cc.encoderConfig.LengthAdjustment
	if cc.encoderConfig.LengthIncludesLengthFieldLength {
		length += cc.encoderConfig.LengthFieldLength
	}

	if length < 0 {
		return nil, ErrTooLessLength
	}
	var out NewDataToCheck

	switch cc.encoderConfig.LengthFieldLength {
	case 1:
		if length >= 256 {
			return nil, fmt.Errorf("length does not fit into a byte: %d", length)
		}
		out = []byte{byte(length)}
	case 2:
		if length >= 65536 {
			return nil, fmt.Errorf("length does not fit into a short integer: %d", length)
		}
		out = make([]byte, 2)
		cc.encoderConfig.ByteOrder.PutUint16(out, uint16(length))
	case 3:
		if length >= 16777216 {
			return nil, fmt.Errorf("length does not fit into a medium integer: %d", length)
		}
		out = writeUint24(cc.encoderConfig.ByteOrder, length)
	case 4:
		out = make([]byte, 4)
		cc.encoderConfig.ByteOrder.PutUint32(out, uint32(length))
	case 8:
		out = make([]byte, 8)
		cc.encoderConfig.ByteOrder.PutUint64(out, uint64(length))
	default:
		return nil, ErrUnsupportedLength
	}
	out =append(cc.encoderConfig.Header,out...)
	out = append(out, buf...)
	if cc.encoderConfig.AddCheckData{
		out=out.AddCheckSum()
		return out,nil
	}
	return out,nil
}

type innerBuffer []byte

func (in *innerBuffer) readN(n int) (buf []byte, err error) {
	if n <= 0 {
		return nil, errors.New("zero or negative length is invalid")
	} else if n > len(*in) {
		return nil, errors.New("exceeding buffer length")
	}
	buf = (*in)[:n]
	*in = (*in)[n:]
	return
}

// Decode ...
func (cc *LengthFieldBasedFrameCodec) Decode(c gnet.Conn) ([]byte, error) {
	var (
		in     innerBuffer
		header []byte
		err    error
	)
	in = c.Read()
	if cc.decoderConfig.LengthFieldOffset > 0 { //discard header(offset)
		header, err = in.readN(cc.decoderConfig.LengthFieldOffset)
		if err != nil {
			return nil, ErrUnexpectedEOF
		}
	}
	if cc.decoderConfig.FinalBytesToStrip < 0 {
		return nil, errors.New("zero or negative length is invalid")
	}

	lenBuf, frameLength, err := cc.getUnadjustedFrameLength(&in)
	if err != nil {
		return nil, err
	}

	// real message length
	msgLength := int(frameLength) + cc.decoderConfig.LengthAdjustment
	msg, err := in.readN(msgLength)
	if err != nil {
		return nil, ErrUnexpectedEOF
	}
	var  fullMessage NewDataToCheck =  make([]byte, len(header)+len(lenBuf)+msgLength)
	//fullMessage := make([]byte, len(header)+len(lenBuf)+msgLength)
	copy(fullMessage, header)
	copy(fullMessage[len(header):], lenBuf)
	copy(fullMessage[len(header)+len(lenBuf):], msg)
	c.ShiftN(len(fullMessage))
	if cc.decoderConfig.CheckData {
		if !fullMessage.CheckData() {
			return nil, errors.New("CheckData failed")
		}
	}
	return fullMessage[cc.decoderConfig.InitialBytesToStrip:(len(fullMessage)-cc.decoderConfig.FinalBytesToStrip)], nil
}

func (cc *LengthFieldBasedFrameCodec) getUnadjustedFrameLength(in *innerBuffer) ([]byte, uint64, error) {
	switch cc.decoderConfig.LengthFieldLength {
	case 1:
		b, err := in.readN(1)
		if err != nil {
			return nil, 0, ErrUnexpectedEOF
		}
		return b, uint64(b[0]), nil
	case 2:
		lenBuf, err := in.readN(2)
		if err != nil {
			return nil, 0, ErrUnexpectedEOF
		}
		return lenBuf, uint64(cc.decoderConfig.ByteOrder.Uint16(lenBuf)), nil
	case 3:
		lenBuf, err := in.readN(3)
		if err != nil {
			return nil, 0, ErrUnexpectedEOF
		}
		return lenBuf, readUint24(cc.decoderConfig.ByteOrder, lenBuf), nil
	case 4:
		lenBuf, err := in.readN(4)
		if err != nil {
			return nil, 0, ErrUnexpectedEOF
		}
		return lenBuf, uint64(cc.decoderConfig.ByteOrder.Uint32(lenBuf)), nil
	case 8:
		lenBuf, err := in.readN(8)
		if err != nil {
			return nil, 0, ErrUnexpectedEOF
		}
		return lenBuf, cc.decoderConfig.ByteOrder.Uint64(lenBuf), nil
	default:
		return nil, 0, ErrUnsupportedLength
	}
}

func readUint24(byteOrder binary.ByteOrder, b []byte) uint64 {
	_ = b[2]
	if byteOrder == binary.LittleEndian {
		return uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16
	}
	return uint64(b[2]) | uint64(b[1])<<8 | uint64(b[0])<<16
}

func writeUint24(byteOrder binary.ByteOrder, v int) []byte {
	b := make([]byte, 3)
	if byteOrder == binary.LittleEndian {
		b[0] = byte(v)
		b[1] = byte(v >> 8)
		b[2] = byte(v >> 16)
	} else {
		b[2] = byte(v)
		b[1] = byte(v >> 8)
		b[0] = byte(v >> 16)
	}
	return b
}

//Recive data check sum
