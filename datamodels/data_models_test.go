// Copyright 2023 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package datamodels

import (
	"bytes"
	"fmt"
)

func (gotFldm *FixedLayoutDataModel) Compare(wantFldm *FixedLayoutDataModel) string {

	if len(wantFldm.Values) != len(wantFldm.Names) {
		return fmt.Sprintf("inconsistent reference: len(Values) %d != %d len(Names)", len(wantFldm.Values), len(wantFldm.Names))
	}

	buf := &bytes.Buffer{}

	if len(wantFldm.Values) != len(gotFldm.Values) {
		fmt.Fprintf(buf, "\nlen(Values): want: %d got: %d", len(wantFldm.Values), len(gotFldm.Values))
	} else {
		for i, wantValue := range wantFldm.Values {
			gotValue := gotFldm.Values[i]
			if wantValue != gotValue {
				fmt.Fprintf(buf, "\nValues[%d]: want: %q got: %q", i, wantValue, gotValue)
			}
		}
	}

	if len(wantFldm.Names) != len(gotFldm.Names) {
		return fmt.Sprintf("len(Names): want: %d got: %d", len(wantFldm.Names), len(gotFldm.Names))
	} else {
		for i, wantName := range wantFldm.Names {
			gotName := gotFldm.Names[i]
			if wantName != gotName {
				fmt.Fprintf(buf, "\nNames[%d]: want: %q got: %q", i, wantName, gotName)
			}
		}
	}

	for group, wantIndexRangeList := range wantFldm.Groups {
		gotIndexRangeList, ok := gotFldm.Groups[group]
		if !ok {
			fmt.Fprintf(buf, "\nGroups: missing group %q", group)
			continue
		}
		// Range lists should be compared order independent:
		wantIndexRange := map[IndexRange]bool{}
		gotIndexRange := map[IndexRange]bool{}
		for _, indexRange := range wantIndexRangeList {
			wantIndexRange[indexRange] = true
		}
		for _, indexRange := range gotIndexRangeList {
			gotIndexRange[indexRange] = true
		}
		for indexRange, _ := range wantIndexRange {
			if !gotIndexRange[indexRange] {
				fmt.Fprintf(buf, "\nGroups[%q]: missing range %v", group, indexRange)
			}
		}
		for indexRange, _ := range gotIndexRange {
			if !wantIndexRange[indexRange] {
				fmt.Fprintf(buf, "\nGroups[%q]: unexpected range %v", group, indexRange)
			}
		}
	}

	for group, _ := range gotFldm.Groups {
		if wantFldm.Groups[group] == nil {
			fmt.Fprintf(buf, "\nGroups: unexpected group %q", group)
		}
	}

	return buf.String()
}

func (gotFcdm *FixedColumnDataModel) Compare(wantFcdm *FixedColumnDataModel) string {
	buf := &bytes.Buffer{}

	for key, wantData := range wantFcdm.Data {
		gotData, ok := gotFcdm.Data[key]
		if !ok {
			fmt.Fprintf(buf, "\nData: missing %q", key)
			continue
		}
		wantValues := wantData.Values
		gotValues := gotData.Values
		if len(wantValues) != len(gotValues) {
			fmt.Fprintf(buf, "\nlen(Data[%q].Values): want: %d got: %d", key, len(wantValues), len(gotValues))
		} else {
			for i, wantValue := range wantValues {
				gotValue := gotValues[i]
				if wantValue != gotValue {
					fmt.Fprintf(buf, "\nData[%q].Values[%d]: want: %q got: %q", key, i, wantValue, gotValue)
				}
			}
		}
		if wantData.ScanID != gotData.ScanID {
			fmt.Fprintf(buf, "\nData[%q].ScanID: want: %d got: %d", key, wantData.ScanID, gotData.ScanID)
		}
	}

	for key := range gotFcdm.Data {
		_, ok := wantFcdm.Data[key]
		if !ok {
			fmt.Fprintf(buf, "\nData: unexpected key: %q", key)
		}
	}

	if wantFcdm.ScanID != gotFcdm.ScanID {
		fmt.Fprintf(buf, "\nScanID: want: %d, got: %d", wantFcdm.ScanID, gotFcdm.ScanID)
	}

	if len(wantFcdm.Columns) != len(gotFcdm.Columns) {
		fmt.Fprintf(buf, "\nlen(Columns): want: %d got: %d", len(wantFcdm.Columns), len(gotFcdm.Columns))
	} else {
		for i, wantColumn := range wantFcdm.Columns {
			gotColumn := gotFcdm.Columns[i]
			if wantColumn != gotColumn {
				fmt.Fprintf(buf, "\nColumns[%d]: want: %q got: %q", i, wantColumn, gotColumn)
			}
		}
	}

	for key, wantInfo := range wantFcdm.Info {
		gotInfo, ok := gotFcdm.Info[key]
		if !ok {
			fmt.Fprintf(buf, "\nInfo: missing %q", key)
		} else if wantInfo != gotInfo {
			fmt.Fprintf(buf, "\nInfo[%q]: want: %q, got: %q", key, wantInfo, gotInfo)
		}
	}

	for key, _ := range gotFcdm.Info {
		_, ok := wantFcdm.Info[key]
		if !ok {
			fmt.Fprintf(buf, "\nInfo: unexpected key: %q", key)
		}
	}

	return buf.String()
}
