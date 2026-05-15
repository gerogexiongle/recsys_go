package rankengine

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// FMTrans maps FM feature field id -> max slot count (same semantics as legacy FMTransData).
type FMTrans struct {
	AllFieldCount int
	FieldTrans    map[int]int
}

// LoadFMTrans parses the legacy trans file: first line total field count, then "field count" per line.
func LoadFMTrans(path string) (*FMTrans, error) {
	if path == "" {
		return &FMTrans{FieldTrans: make(map[int]int)}, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	if !sc.Scan() {
		return nil, fmt.Errorf("fm trans: empty file")
	}
	all, err := strconv.Atoi(strings.TrimSpace(sc.Text()))
	if err != nil {
		return nil, fmt.Errorf("fm trans first line: %w", err)
	}
	ft := make(map[int]int)
	sum := 0
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) != 2 {
			return nil, fmt.Errorf("fm trans bad line %q", line)
		}
		field, _ := strconv.Atoi(parts[0])
		cnt, _ := strconv.Atoi(parts[1])
		if field <= 0 || cnt < 0 {
			return nil, fmt.Errorf("fm trans invalid field/count %q", line)
		}
		ft[field] = cnt
		sum += cnt
	}
	if sum != all {
		return nil, fmt.Errorf("fm trans sum %d != all_field_count %d", sum, all)
	}
	return &FMTrans{AllFieldCount: all, FieldTrans: ft}, nil
}
