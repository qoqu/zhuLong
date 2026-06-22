// 语音交互 - STT/TTS/语音模式
package voice

// STTProvider 语音转文本提供者
type STTProvider interface {
	Name() string
	Transcribe(audioData []byte) (string, error)
}

// TTSProvider 文本转语音提供者
type TTSProvider interface {
	Name() string
	Synthesize(text string) ([]byte, error)
}

// Engine 语音引擎
type Engine struct {
	stt STTProvider
	tts TTSProvider
}

func NewEngine() *Engine { return &Engine{} }

func (e *Engine) SetSTT(p STTProvider) { e.stt = p }
func (e *Engine) SetTTS(p TTSProvider) { e.tts = p }

func (e *Engine) Transcribe(data []byte) (string, error) {
	if e.stt == nil {
		return "", nil
	}
	return e.stt.Transcribe(data)
}

func (e *Engine) Synthesize(text string) ([]byte, error) {
	if e.tts == nil {
		return nil, nil
	}
	return e.tts.Synthesize(text)
}

func (e *Engine) HasSTT() bool { return e.stt != nil }
func (e *Engine) HasTTS() bool { return e.tts != nil }
