// Common definitions for all parsers

package procfs

// There are 2 use cases for parsed data: as-is and involved in some
// calculations/further processing. The data source is mainly /proc file system,
// so the source is text. The generated metrics as in Prometheus exposition
// format, which is again text. Thus the most efficient parsed format for as-is
// data is []byte where the slice is against the full content of the file,
// which, for efficiency purposes is read in one go. A parsed datum is defined
// by start:end offsets in the content buffer.
type SliceOffsets struct {
	Start, End int
}

// Most files consist of words delimited by white spaces; the file content is
// scanned one byte at the time and the following arrays provide a convenient
// lookup for deciding if a byte is a whitespace or not:
var isWhitespace = [256]bool{
	' ':  true,
	'\t': true,
}

var isWhitespaceNl = [256]bool{
	' ':  true,
	'\t': true,
	'\n': true,
}

func getCurrentLine(buf []byte, pos int) string {
	var lineStart, lineEnd int
	l := len(buf)
	if pos < 0 {
		lineStart, lineEnd = -pos, -pos
		for ; lineStart > 0 && buf[lineStart-1] != '\n'; lineStart-- {
		}
	} else {
		lineStart, lineEnd = pos, pos
	}
	for ; lineEnd < l && buf[lineEnd] != '\n'; lineEnd++ {
	}
	return string(buf[lineStart:lineEnd])
}
