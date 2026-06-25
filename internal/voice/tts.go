// 本地 TTS provider（使用系统命令）
package voice

import (
	"fmt"
	"os/exec"
	"runtime"
)

// LocalTTS 使用系统命令的 TTS provider
type LocalTTS struct {
	voice string
}

// NewLocalTTS creates a local TTS provider
func NewLocalTTS() *LocalTTS {
	return &LocalTTS{voice: "default"}
}

func (t *LocalTTS) Name() string { return "local-tts" }

// Synthesize 使用系统命令合成语音
// Windows: PowerShell SpeechSynthesis
// macOS: say 命令
// Linux: espeak
func (t *LocalTTS) Synthesize(text string) ([]byte, error) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		// 使用 PowerShell 的 SpeechSynthesis
		script := fmt.Sprintf(`
Add-Type -AssemblyName System.Speech
$synth = New-Object System.Speech.Synthesis.SpeechSynthesizer
$synth.Speak("%s")
`, text)
		cmd = exec.Command("powershell", "-Command", script)
	case "darwin":
		cmd = exec.Command("say", text)
	case "linux":
		cmd = exec.Command("espeak", text)
	default:
		return nil, fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("TTS failed: %w", err)
	}

	// 本地 TTS 不返回音频数据（直接播放）
	return nil, nil
}

// Speak 使用系统命令直接播放语音（不返回音频数据）
func (t *LocalTTS) Speak(text string) error {
	_, err := t.Synthesize(text)
	return err
}
