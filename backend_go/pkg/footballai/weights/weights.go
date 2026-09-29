// Package weights implements the custom FootballMoE model format (.fmoe)
// and training checkpoints (.fmckpt). The format is little-endian, fully
// self-describing, checksummed, and carries architecture metadata so
// incompatible weights are rejected at load time.
package weights

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"

	"football_sim/pkg/footballai"
)

func init() {
	// Register this package as the .fmoe loader for footballai.Brain. The
	// server binary imports weights, so the hook is always installed.
	footballai.RegisterModelLoader(Load)
}

// Format constants.
const (
	Magic                 = "FBMOE001"
	FormatVersion  uint16 = 1
	DTypeFP32      uint8  = 0
	FlagCheckpoint uint32 = 1 << 0
	ChecksumSHA256 uint8  = 1

	headerSize = 2 + 4 + 1 + 4 + 4 + 8 + 4 + 4 + 4 + 4 + 8 + 1 + 16
)

// TensorEntry describes one tensor in the model directory. Tensors are
// addressed by name; consumers must not depend on order.
type TensorEntry struct {
	Name   string
	DType  uint8
	Rank   uint8
	Dims   []uint32
	Offset uint64
	Length uint64 // element count
}

// Inspection is everything modeltool needs without building the network.
type Inspection struct {
	FormatVersion  uint16
	ModelVersion   uint32
	DType          uint8
	Flags          uint32
	TensorCount    uint32
	ParameterCount uint64
	Config         footballai.ModelConfig
	Norm           footballai.NormMeta
	Tensors        []TensorEntry
	ModelHash      string
	ChecksumOK     bool
	DataHash       string // SHA-256 over the tensor data section
}

// CheckpointState is the resumable training state stored in .fmckpt files.
// Runtime inference never requires it.
type CheckpointState struct {
	Epoch        int                `json:"epoch"`
	Step         int                `json:"step"`
	LearningRate float32            `json:"learning_rate"`
	BestValScore float32            `json:"best_val_score"`
	Metrics      map[string]float64 `json:"metrics,omitempty"`
	RandomSeed   int64              `json:"random_seed"`
	// Moments maps parameter names to Adam first/second moments. Serialized
	// as tensors named optim.m.<param> / optim.v.<param>.
	Moments map[string]Moment
}

// Moment holds AdamW state for one parameter.
type Moment struct {
	M []float32
	V []float32
}

type fileHeader struct {
	FormatVersion   uint16
	ModelVersion    uint32
	DType           uint8
	Flags           uint32
	TensorCount     uint32
	ParameterCount  uint64
	ConfigLength    uint32
	NormLength      uint32
	ExtraLength     uint32
	TensorTblLength uint32
	TensorDataLen   uint64
	ChecksumType    uint8
}

type namedTensor struct {
	name string
	dims []uint32
	data []float32
}

// Save exports a network as an immutable deployment .fmoe file and returns
// the model hash (SHA-256 over the tensor data section).
func Save(path string, net *footballai.Network) (string, error) {
	return writeModel(path, net, nil, 0)
}

// SaveCheckpoint exports a training checkpoint with optimizer state.
func SaveCheckpoint(path string, net *footballai.Network, state *CheckpointState) (string, error) {
	return writeModel(path, net, state, FlagCheckpoint)
}

func writeModel(path string, net *footballai.Network, state *CheckpointState, flags uint32) (string, error) {
	cfg := net.Config()
	if err := cfg.Validate(); err != nil {
		return "", err
	}
	norm := net.NormMeta()

	cfgJSON, err := json.Marshal(cfg)
	if err != nil {
		return "", fmt.Errorf("weights: marshal config: %w", err)
	}
	normJSON, err := json.Marshal(norm)
	if err != nil {
		return "", fmt.Errorf("weights: marshal norm: %w", err)
	}
	var extraJSON []byte
	var moments map[string]Moment
	if state != nil {
		moments = state.Moments
		state.Moments = nil // moments travel as tensors, not JSON
		extraJSON, err = json.Marshal(state)
		state.Moments = moments // restore the caller's copy
		if err != nil {
			return "", fmt.Errorf("weights: marshal checkpoint state: %w", err)
		}
	}

	// Collect tensors.
	var tensors []namedTensor
	paramCount := uint64(0)
	for _, p := range net.Params() {
		tensors = append(tensors, namedTensor{name: p.Name, dims: []uint32{uint32(len(p.Data))}, data: p.Data})
		paramCount += uint64(len(p.Data))
	}
	if state != nil {
		names := make([]string, 0, len(moments))
		for name := range moments {
			names = append(names, name)
		}
		sortStrings(names)
		for _, name := range names {
			m := moments[name]
			tensors = append(tensors,
				namedTensor{name: "optim.m." + name, dims: []uint32{uint32(len(m.M))}, data: m.M},
				namedTensor{name: "optim.v." + name, dims: []uint32{uint32(len(m.V))}, data: m.V})
		}
	}

	// Serialize the tensor table and data.
	var table bytes.Buffer
	var data bytes.Buffer
	for _, t := range tensors {
		offset := uint64(data.Len())
		for _, v := range t.data {
			var buf [4]byte
			binary.LittleEndian.PutUint32(buf[:], math.Float32bits(v))
			data.Write(buf[:])
		}
		var nameBuf [2]byte
		binary.LittleEndian.PutUint16(nameBuf[:], uint16(len(t.name)))
		table.Write(nameBuf[:])
		table.WriteString(t.name)
		table.WriteByte(DTypeFP32)
		table.WriteByte(uint8(len(t.dims)))
		for _, d := range t.dims {
			var buf [4]byte
			binary.LittleEndian.PutUint32(buf[:], d)
			table.Write(buf[:])
		}
		var buf8 [8]byte
		binary.LittleEndian.PutUint64(buf8[:], offset)
		table.Write(buf8[:])
		binary.LittleEndian.PutUint64(buf8[:], uint64(len(t.data)))
		table.Write(buf8[:])
	}

	h := fileHeader{
		FormatVersion:   FormatVersion,
		ModelVersion:    cfg.ModelVersion,
		DType:           DTypeFP32,
		Flags:           flags,
		TensorCount:     uint32(len(tensors)),
		ParameterCount:  paramCount,
		ConfigLength:    uint32(len(cfgJSON)),
		NormLength:      uint32(len(normJSON)),
		ExtraLength:     uint32(len(extraJSON)),
		TensorTblLength: uint32(table.Len()),
		TensorDataLen:   uint64(data.Len()),
		ChecksumType:    ChecksumSHA256,
	}

	var out bytes.Buffer
	out.WriteString(Magic)
	writeHeader(&out, &h)
	out.Write(cfgJSON)
	out.Write(normJSON)
	out.Write(extraJSON)
	out.Write(table.Bytes())
	out.Write(data.Bytes())

	sum := sha256.Sum256(out.Bytes())
	out.Write(sum[:])

	dataHash := sha256.Sum256(data.Bytes())
	modelHash := fmt.Sprintf("%x", dataHash)

	f, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("weights: create %s: %w", path, err)
	}
	defer f.Close()
	if _, err := f.Write(out.Bytes()); err != nil {
		return "", fmt.Errorf("weights: write %s: %w", path, err)
	}
	return modelHash, nil
}

func writeHeader(w io.Writer, h *fileHeader) {
	var b [8]byte
	binary.LittleEndian.PutUint16(b[:2], h.FormatVersion)
	_, _ = w.Write(b[:2])
	binary.LittleEndian.PutUint32(b[:4], h.ModelVersion)
	_, _ = w.Write(b[:4])
	_, _ = w.Write([]byte{h.DType})
	binary.LittleEndian.PutUint32(b[:4], h.Flags)
	_, _ = w.Write(b[:4])
	binary.LittleEndian.PutUint32(b[:4], h.TensorCount)
	_, _ = w.Write(b[:4])
	binary.LittleEndian.PutUint64(b[:8], h.ParameterCount)
	_, _ = w.Write(b[:8])
	binary.LittleEndian.PutUint32(b[:4], h.ConfigLength)
	_, _ = w.Write(b[:4])
	binary.LittleEndian.PutUint32(b[:4], h.NormLength)
	_, _ = w.Write(b[:4])
	binary.LittleEndian.PutUint32(b[:4], h.ExtraLength)
	_, _ = w.Write(b[:4])
	binary.LittleEndian.PutUint32(b[:4], h.TensorTblLength)
	_, _ = w.Write(b[:4])
	binary.LittleEndian.PutUint64(b[:8], h.TensorDataLen)
	_, _ = w.Write(b[:8])
	_, _ = w.Write([]byte{h.ChecksumType})
	_, _ = w.Write(make([]byte, 16)) // reserved
}

func readHeader(r io.Reader) (*fileHeader, error) {
	raw := make([]byte, headerSize)
	if _, err := io.ReadFull(r, raw); err != nil {
		return nil, fmt.Errorf("weights: truncated header: %w", err)
	}
	h := &fileHeader{}
	h.FormatVersion = binary.LittleEndian.Uint16(raw[0:2])
	h.ModelVersion = binary.LittleEndian.Uint32(raw[2:6])
	h.DType = raw[6]
	h.Flags = binary.LittleEndian.Uint32(raw[7:11])
	h.TensorCount = binary.LittleEndian.Uint32(raw[11:15])
	h.ParameterCount = binary.LittleEndian.Uint64(raw[15:23])
	h.ConfigLength = binary.LittleEndian.Uint32(raw[23:27])
	h.NormLength = binary.LittleEndian.Uint32(raw[27:31])
	h.ExtraLength = binary.LittleEndian.Uint32(raw[31:35])
	h.TensorTblLength = binary.LittleEndian.Uint32(raw[35:39])
	h.TensorDataLen = binary.LittleEndian.Uint64(raw[39:47])
	h.ChecksumType = raw[47]
	return h, nil
}

// readFile parses a model or checkpoint file into its raw parts.
func readFile(path string) (hdr *fileHeader, cfg footballai.ModelConfig, norm footballai.NormMeta,
	extra []byte, tensors []TensorEntry, data []byte, checksumOK bool, err error) {

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, cfg, norm, nil, nil, nil, false, fmt.Errorf("weights: read %s: %w", path, err)
	}
	if len(raw) < 8+headerSize+32 {
		return nil, cfg, norm, nil, nil, nil, false, fmt.Errorf("weights: file too small")
	}
	if string(raw[:8]) != Magic {
		return nil, cfg, norm, nil, nil, nil, false, fmt.Errorf("weights: bad magic %q, expected %q", string(raw[:8]), Magic)
	}
	body := raw[8:]
	stored := body[len(body)-32:]
	body = body[:len(body)-32]
	sum := sha256.Sum256(append([]byte(Magic), body...))
	checksumOK = bytes.Equal(sum[:], stored)

	r := bytes.NewReader(body)
	hdr, err = readHeader(r)
	if err != nil {
		return nil, cfg, norm, nil, nil, nil, false, err
	}
	if hdr.FormatVersion != FormatVersion {
		return nil, cfg, norm, nil, nil, nil, false, fmt.Errorf("weights: format version %d unsupported (expected %d)", hdr.FormatVersion, FormatVersion)
	}
	if hdr.DType != DTypeFP32 {
		return nil, cfg, norm, nil, nil, nil, false, fmt.Errorf("weights: dtype %d unsupported (expected FP32)", hdr.DType)
	}

	cfgBuf := make([]byte, hdr.ConfigLength)
	if _, err := io.ReadFull(r, cfgBuf); err != nil {
		return nil, cfg, norm, nil, nil, nil, false, fmt.Errorf("weights: truncated config: %w", err)
	}
	if err := json.Unmarshal(cfgBuf, &cfg); err != nil {
		return nil, cfg, norm, nil, nil, nil, false, fmt.Errorf("weights: bad config JSON: %w", err)
	}
	normBuf := make([]byte, hdr.NormLength)
	if _, err := io.ReadFull(r, normBuf); err != nil {
		return nil, cfg, norm, nil, nil, nil, false, fmt.Errorf("weights: truncated norm: %w", err)
	}
	if len(normBuf) > 0 {
		if err := json.Unmarshal(normBuf, &norm); err != nil {
			return nil, cfg, norm, nil, nil, nil, false, fmt.Errorf("weights: bad norm JSON: %w", err)
		}
	}
	if hdr.ExtraLength > 0 {
		extra = make([]byte, hdr.ExtraLength)
		if _, err := io.ReadFull(r, extra); err != nil {
			return nil, cfg, norm, nil, nil, nil, false, fmt.Errorf("weights: truncated extra: %w", err)
		}
	}

	table := make([]byte, hdr.TensorTblLength)
	if _, err := io.ReadFull(r, table); err != nil {
		return nil, cfg, norm, nil, nil, nil, false, fmt.Errorf("weights: truncated tensor table: %w", err)
	}
	tr := bytes.NewReader(table)
	tensors = make([]TensorEntry, 0, hdr.TensorCount)
	for i := uint32(0); i < hdr.TensorCount; i++ {
		var nameLen [2]byte
		if _, err := io.ReadFull(tr, nameLen[:]); err != nil {
			return nil, cfg, norm, nil, nil, nil, false, fmt.Errorf("weights: truncated tensor entry: %w", err)
		}
		name := make([]byte, binary.LittleEndian.Uint16(nameLen[:]))
		if _, err := io.ReadFull(tr, name); err != nil {
			return nil, cfg, norm, nil, nil, nil, false, fmt.Errorf("weights: truncated tensor name: %w", err)
		}
		meta := make([]byte, 2) // dtype, rank
		if _, err := io.ReadFull(tr, meta); err != nil {
			return nil, cfg, norm, nil, nil, nil, false, fmt.Errorf("weights: truncated tensor meta: %w", err)
		}
		dims := make([]uint32, meta[1])
		for d := range dims {
			var b4 [4]byte
			if _, err := io.ReadFull(tr, b4[:]); err != nil {
				return nil, cfg, norm, nil, nil, nil, false, fmt.Errorf("weights: truncated dims: %w", err)
			}
			dims[d] = binary.LittleEndian.Uint32(b4[:])
		}
		var b8 [8]byte
		if _, err := io.ReadFull(tr, b8[:]); err != nil {
			return nil, cfg, norm, nil, nil, nil, false, fmt.Errorf("weights: truncated offset: %w", err)
		}
		offset := binary.LittleEndian.Uint64(b8[:])
		if _, err := io.ReadFull(tr, b8[:]); err != nil {
			return nil, cfg, norm, nil, nil, nil, false, fmt.Errorf("weights: truncated length: %w", err)
		}
		length := binary.LittleEndian.Uint64(b8[:])
		tensors = append(tensors, TensorEntry{
			Name: string(name), DType: meta[0], Rank: meta[1], Dims: dims,
			Offset: offset, Length: length,
		})
	}
	data = make([]byte, hdr.TensorDataLen)
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, cfg, norm, nil, nil, nil, false, fmt.Errorf("weights: truncated tensor data: %w", err)
	}
	if r.Len() != 0 {
		return nil, cfg, norm, nil, nil, nil, false, fmt.Errorf("weights: %d trailing bytes", r.Len())
	}
	return hdr, cfg, norm, extra, tensors, data, checksumOK, nil
}

// Verify checks a model file's checksum and structural integrity without
// building the network.
func Verify(path string) error {
	hdr, cfg, _, _, tensors, _, ok, err := readFile(path)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("weights: checksum mismatch")
	}
	if err := cfg.Validate(); err != nil {
		return err
	}
	// Cross-check the tensor directory against the data section.
	var maxEnd uint64
	for _, t := range tensors {
		end := t.Offset + 4*t.Length // offsets are bytes, lengths are elements
		if end > maxEnd {
			maxEnd = end
		}
	}
	if maxEnd != hdr.TensorDataLen {
		return fmt.Errorf("weights: tensor directory (%d bytes) does not cover data section (%d bytes)", maxEnd, hdr.TensorDataLen)
	}
	return nil
}

// Inspect returns file metadata without instantiating the model.
func Inspect(path string) (*Inspection, error) {
	hdr, cfg, norm, _, tensors, data, ok, err := readFile(path)
	if err != nil {
		return nil, err
	}
	dataHash := sha256.Sum256(data)
	return &Inspection{
		FormatVersion:  hdr.FormatVersion,
		ModelVersion:   hdr.ModelVersion,
		DType:          hdr.DType,
		Flags:          hdr.Flags,
		TensorCount:    hdr.TensorCount,
		ParameterCount: hdr.ParameterCount,
		Config:         cfg,
		Norm:           norm,
		Tensors:        tensors,
		ChecksumOK:     ok,
		DataHash:       fmt.Sprintf("%x", dataHash),
	}, nil
}

// Load reads a .fmoe file and returns the immutable runtime model.
func Load(path string) (*footballai.Model, error) {
	hdr, cfg, norm, extra, tensors, data, ok, err := readFile(path)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("weights: checksum mismatch in %s", path)
	}
	if hdr.Flags&FlagCheckpoint != 0 {
		return nil, fmt.Errorf("weights: %s is a training checkpoint; load a .fmoe deployment model", path)
	}
	if len(extra) != 0 {
		return nil, fmt.Errorf("weights: unexpected extra section in deployment model")
	}
	net, err := buildNetwork(cfg, norm, tensors, data)
	if err != nil {
		return nil, err
	}
	dataHash := sha256.Sum256(data)
	return footballai.BuildModel(net, fmt.Sprintf("%x", dataHash)), nil
}

// LoadCheckpoint reads a .fmckpt and returns the model plus resumable state.
func LoadCheckpoint(path string) (*footballai.Model, *CheckpointState, error) {
	hdr, cfg, norm, extra, tensors, data, ok, err := readFile(path)
	if err != nil {
		return nil, nil, err
	}
	if !ok {
		return nil, nil, fmt.Errorf("weights: checksum mismatch in %s", path)
	}
	if hdr.Flags&FlagCheckpoint == 0 {
		return nil, nil, fmt.Errorf("weights: %s is not a checkpoint", path)
	}
	state := &CheckpointState{Moments: map[string]Moment{}}
	if len(extra) > 0 {
		if err := json.Unmarshal(extra, state); err != nil {
			return nil, nil, fmt.Errorf("weights: bad checkpoint state: %w", err)
		}
	}
	state.Moments = map[string]Moment{}
	for _, t := range tensors {
		if len(t.Name) > 8 && t.Name[:8] == "optim.m." {
			name := t.Name[8:]
			m := state.Moments[name]
			m.M = readTensor(t, data)
			state.Moments[name] = m
		} else if len(t.Name) > 8 && t.Name[:8] == "optim.v." {
			name := t.Name[8:]
			m := state.Moments[name]
			m.V = readTensor(t, data)
			state.Moments[name] = m
		}
	}
	// Model tensors only.
	var modelTensors []TensorEntry
	for _, t := range tensors {
		if len(t.Name) > 8 && (t.Name[:8] == "optim.m." || t.Name[:8] == "optim.v.") {
			continue
		}
		modelTensors = append(modelTensors, t)
	}
	net, err := buildNetwork(cfg, norm, modelTensors, data)
	if err != nil {
		return nil, nil, err
	}
	dataHash := sha256.Sum256(data)
	return footballai.BuildModel(net, fmt.Sprintf("%x", dataHash)), state, nil
}

func buildNetwork(cfg footballai.ModelConfig, norm footballai.NormMeta, tensors []TensorEntry, data []byte) (*footballai.Network, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	net, err := footballai.NewNetwork(cfg, 1)
	if err != nil {
		return nil, err
	}
	if len(norm.Slots) != 0 {
		if err := net.SetNormMeta(norm); err != nil {
			return nil, err
		}
	}
	named := make(map[string][]float32, len(tensors))
	for _, t := range tensors {
		named[t.Name] = readTensor(t, data)
	}
	if err := net.ApplyParams(named); err != nil {
		return nil, err
	}
	return net, nil
}

func readTensor(t TensorEntry, data []byte) []float32 {
	out := make([]float32, t.Length)
	if t.Offset+4*t.Length > uint64(len(data)) {
		return out // callers reject via ApplyParams length mismatch
	}
	for i := uint64(0); i < t.Length; i++ {
		out[i] = math.Float32frombits(binary.LittleEndian.Uint32(data[t.Offset+4*i:]))
	}
	return out
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
