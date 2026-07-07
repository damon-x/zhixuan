package chunker

import "zhixuan/server/config"

// Chunk represents a text slice with its index.
type Chunk struct {
	Text  string `json:"text"`
	Index int    `json:"index"`
}

// Split splits text into overlapping chunks using a sliding window.
// It operates on []rune to properly handle Chinese characters.
func Split(text string) []Chunk {
	runes := []rune(text)
	windowSize := config.ChunkWindowSize
	overlap := config.ChunkOverlap

	if len(runes) <= windowSize {
		return []Chunk{{Text: text, Index: 0}}
	}

	var chunks []Chunk
	step := windowSize - overlap
	for start := 0; start < len(runes); start += step {
		end := start + windowSize
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, Chunk{
			Text:  string(runes[start:end]),
			Index: len(chunks),
		})
		if end == len(runes) {
			break
		}
	}
	return chunks
}
