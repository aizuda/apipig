package toolkit

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"strconv"
)

// HexToDecimal 16 进制字节数组转为 10 进制
func HexToDecimal(bytes []byte) (int64, error) {
	return HexStrToDecimal(string(bytes))
}

// HexStrToDecimal 16 进制字节数组转为 10 进制
func HexStrToDecimal(str string) (int64, error) {
	decimal, err := strconv.ParseInt(str, 16, 64)
	if err != nil {
		return 0, err
	}
	return decimal, nil
}

// HexToASCII 16 进制字节数组转为 ASCII 码
func HexToASCII(hexBytes []byte) (string, error) {
	asciiString := make([]byte, hex.DecodedLen(len(hexBytes)))
	n, err := hex.Decode(asciiString, hexBytes)
	if err != nil {
		return "", err
	}
	return string(asciiString[:n]), nil
}

func BCDToTime(data []byte) string {
	hexStr := string(data)
	return fmt.Sprintf("20%s:%s:%s %s:%s:%s", hexStr[10:12], hexStr[8:10],
		hexStr[6:8], hexStr[4:6], hexStr[2:4], hexStr[0:2])
}

func Percent16LE(data []byte) float32 {
	l := ReadHexStr16LE(data)
	if l > 0 {
		return float32(l) * 0.01
	}
	return 0
}

func ReadHexStr16LE(data []byte) uint16 {
	bts, _ := hex.DecodeString(string(data))
	return ReadUint16LE(bts)
}

// ReadUint16LE 读取 2 字节并解析为 uint16
func ReadUint16LE(data []byte) uint16 {
	return binary.LittleEndian.Uint16(data)
}

// ReadUint32LE 读取 4 字节并解析为 uint32
func ReadUint32LE(data []byte) uint32 {
	return binary.LittleEndian.Uint32(data)
}

// ReadUint64LE 读取 8 字节并解析为 uint64
func ReadUint64LE(data []byte) uint64 {
	return binary.LittleEndian.Uint64(data)
}

func HexToFloat32LE(data []byte) (float32, error) {
	// 将十六进制字符串转换为字节数组
	bts, err := hex.DecodeString(string(data))
	if err != nil {
		return 0, err
	}

	// 检查字节数组长度是否为 4
	if len(bts) != 4 {
		return 0, fmt.Errorf("invalid byte length, expected 4 bytes, got %d bytes", len(bts))
	}

	val := ReadUint32LE(bts)
	if val > 0 {
		// 将 uint32 解析为 float32
		return math.Float32frombits(val), nil
	}
	return 0, nil
}

// HexToTimestamp 将 32 位小端序十六进制字符串转换为时间戳
func HexToTimestamp(data []byte) (uint32, error) {
	// 将十六进制字符串转换为字节数组
	bts, err := hex.DecodeString(string(data))
	if err != nil {
		return 0, err
	}

	// 检查字节数组长度是否为 4
	if len(bts) != 4 {
		return 0, fmt.Errorf("invalid byte length, expected 4 bytes, got %d bytes", len(bts))
	}

	// 以小端序读取字节数组并转换为 uint32
	return ReadUint32LE(bts), nil
}

// Uint16ToLittleEndianHex 将 uint16 转换为小端序的十六进制字符串
func Uint16ToLittleEndianHex(num uint16) (string, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, num)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", buf.Bytes()), nil
}
