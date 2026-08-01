package api

func terminalChunkEnd(data []byte, start, limit int) int {
	candidate := min(start+limit, len(data))
	end := candidate
	for end < len(data) && end > start && data[end]&0xc0 == 0x80 {
		end--
	}
	if end == start {
		return candidate
	}
	return end
}
