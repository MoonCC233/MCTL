package minecraft

import (
	"encoding/binary"
	"fmt"
	"io"
	"sort"
)

const (
	nbtEnd       byte = 0
	nbtByte      byte = 1
	nbtShort     byte = 2
	nbtInt       byte = 3
	nbtLong      byte = 4
	nbtFloat     byte = 5
	nbtDouble    byte = 6
	nbtByteArray byte = 7
	nbtString    byte = 8
	nbtList      byte = 9
	nbtCompound  byte = 10
	nbtIntArray  byte = 11
	nbtLongArray byte = 12
)

type nbtListValue struct {
	ElementType byte
	Values      []any
}

func readNBT(r io.Reader) (string, map[string]any, error) {
	tag, err := readNBTByte(r)
	if err != nil {
		return "", nil, err
	}
	if tag != nbtCompound {
		return "", nil, fmt.Errorf("NBT 根标签不是 compound: %d", tag)
	}
	name, err := readNBTString(r)
	if err != nil {
		return "", nil, err
	}
	value, err := readNBTPayload(r, nbtCompound)
	if err != nil {
		return "", nil, err
	}
	root, ok := value.(map[string]any)
	if !ok {
		return "", nil, fmt.Errorf("NBT 根 compound 无效")
	}
	return name, root, nil
}

func writeNBT(w io.Writer, name string, root map[string]any) error {
	if err := writeNBTByte(w, nbtCompound); err != nil {
		return err
	}
	if err := writeNBTString(w, name); err != nil {
		return err
	}
	return writeNBTPayload(w, nbtCompound, root)
}

func readNBTPayload(r io.Reader, tag byte) (any, error) {
	switch tag {
	case nbtByte:
		v, err := readNBTByte(r)
		return int8(v), err
	case nbtShort:
		var v int16
		err := binary.Read(r, binary.BigEndian, &v)
		return v, err
	case nbtInt:
		var v int32
		err := binary.Read(r, binary.BigEndian, &v)
		return v, err
	case nbtLong:
		var v int64
		err := binary.Read(r, binary.BigEndian, &v)
		return v, err
	case nbtFloat:
		var v float32
		err := binary.Read(r, binary.BigEndian, &v)
		return v, err
	case nbtDouble:
		var v float64
		err := binary.Read(r, binary.BigEndian, &v)
		return v, err
	case nbtByteArray:
		n, err := readNBTInt(r)
		if err != nil {
			return nil, err
		}
		buf := make([]byte, n)
		_, err = io.ReadFull(r, buf)
		return buf, err
	case nbtString:
		return readNBTString(r)
	case nbtList:
		elem, err := readNBTByte(r)
		if err != nil {
			return nil, err
		}
		n, err := readNBTInt(r)
		if err != nil {
			return nil, err
		}
		list := nbtListValue{ElementType: elem}
		for i := int32(0); i < n; i++ {
			v, err := readNBTPayload(r, elem)
			if err != nil {
				return nil, err
			}
			list.Values = append(list.Values, v)
		}
		return list, nil
	case nbtCompound:
		out := map[string]any{}
		for {
			t, err := readNBTByte(r)
			if err != nil {
				return nil, err
			}
			if t == nbtEnd {
				break
			}
			name, err := readNBTString(r)
			if err != nil {
				return nil, err
			}
			v, err := readNBTPayload(r, t)
			if err != nil {
				return nil, err
			}
			out[name] = v
		}
		return out, nil
	case nbtIntArray:
		n, err := readNBTInt(r)
		if err != nil {
			return nil, err
		}
		arr := make([]int32, n)
		err = binary.Read(r, binary.BigEndian, arr)
		return arr, err
	case nbtLongArray:
		n, err := readNBTInt(r)
		if err != nil {
			return nil, err
		}
		arr := make([]int64, n)
		err = binary.Read(r, binary.BigEndian, arr)
		return arr, err
	default:
		return nil, fmt.Errorf("不支持的 NBT 标签: %d", tag)
	}
}

func writeNBTPayload(w io.Writer, tag byte, value any) error {
	switch tag {
	case nbtByte:
		return writeNBTByte(w, byte(value.(int8)))
	case nbtShort:
		return binary.Write(w, binary.BigEndian, value.(int16))
	case nbtInt:
		return binary.Write(w, binary.BigEndian, value.(int32))
	case nbtLong:
		return binary.Write(w, binary.BigEndian, value.(int64))
	case nbtFloat:
		return binary.Write(w, binary.BigEndian, value.(float32))
	case nbtDouble:
		return binary.Write(w, binary.BigEndian, value.(float64))
	case nbtByteArray:
		arr := value.([]byte)
		if err := writeNBTInt(w, int32(len(arr))); err != nil {
			return err
		}
		_, err := w.Write(arr)
		return err
	case nbtString:
		return writeNBTString(w, value.(string))
	case nbtList:
		list := value.(nbtListValue)
		if err := writeNBTByte(w, list.ElementType); err != nil {
			return err
		}
		if err := writeNBTInt(w, int32(len(list.Values))); err != nil {
			return err
		}
		for _, v := range list.Values {
			if err := writeNBTPayload(w, list.ElementType, v); err != nil {
				return err
			}
		}
		return nil
	case nbtCompound:
		obj := value.(map[string]any)
		keys := make([]string, 0, len(obj))
		for k := range obj {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			t, err := nbtTagFor(obj[k])
			if err != nil {
				return fmt.Errorf("%s: %w", k, err)
			}
			if err := writeNBTByte(w, t); err != nil {
				return err
			}
			if err := writeNBTString(w, k); err != nil {
				return err
			}
			if err := writeNBTPayload(w, t, obj[k]); err != nil {
				return err
			}
		}
		return writeNBTByte(w, nbtEnd)
	case nbtIntArray:
		arr := value.([]int32)
		if err := writeNBTInt(w, int32(len(arr))); err != nil {
			return err
		}
		return binary.Write(w, binary.BigEndian, arr)
	case nbtLongArray:
		arr := value.([]int64)
		if err := writeNBTInt(w, int32(len(arr))); err != nil {
			return err
		}
		return binary.Write(w, binary.BigEndian, arr)
	default:
		return fmt.Errorf("不支持的 NBT 标签: %d", tag)
	}
}

func nbtTagFor(v any) (byte, error) {
	switch v.(type) {
	case int8:
		return nbtByte, nil
	case int16:
		return nbtShort, nil
	case int32:
		return nbtInt, nil
	case int64:
		return nbtLong, nil
	case float32:
		return nbtFloat, nil
	case float64:
		return nbtDouble, nil
	case []byte:
		return nbtByteArray, nil
	case string:
		return nbtString, nil
	case nbtListValue:
		return nbtList, nil
	case map[string]any:
		return nbtCompound, nil
	case []int32:
		return nbtIntArray, nil
	case []int64:
		return nbtLongArray, nil
	default:
		return 0, fmt.Errorf("不支持的 NBT 值类型 %T", v)
	}
}

func readNBTByte(r io.Reader) (byte, error) {
	var b [1]byte
	_, err := io.ReadFull(r, b[:])
	return b[0], err
}

func writeNBTByte(w io.Writer, b byte) error {
	_, err := w.Write([]byte{b})
	return err
}

func readNBTString(r io.Reader) (string, error) {
	var n uint16
	if err := binary.Read(r, binary.BigEndian, &n); err != nil {
		return "", err
	}
	buf := make([]byte, n)
	_, err := io.ReadFull(r, buf)
	return string(buf), err
}

func writeNBTString(w io.Writer, s string) error {
	data := []byte(s)
	if len(data) > 65535 {
		return fmt.Errorf("NBT 字符串过长")
	}
	if err := binary.Write(w, binary.BigEndian, uint16(len(data))); err != nil {
		return err
	}
	_, err := w.Write(data)
	return err
}

func readNBTInt(r io.Reader) (int32, error) {
	var n int32
	err := binary.Read(r, binary.BigEndian, &n)
	return n, err
}

func writeNBTInt(w io.Writer, n int32) error {
	return binary.Write(w, binary.BigEndian, n)
}
