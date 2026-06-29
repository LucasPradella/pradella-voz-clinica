import { useEffect, useRef, useState } from 'react'

const MAX_DURATION_S = 120
const WARN_THRESHOLD_S = 100

type RecorderState = 'idle' | 'recording' | 'stopped' | 'error'

interface RecorderProps {
  onAudioReady: (blob: Blob) => void
  disabled?: boolean
}

export function Recorder({ onAudioReady, disabled = false }: RecorderProps) {
  const [state, setState] = useState<RecorderState>('idle')
  const [elapsed, setElapsed] = useState(0)
  const [errorMsg, setErrorMsg] = useState('')

  const mediaRecorderRef = useRef<MediaRecorder | null>(null)
  const chunksRef = useRef<Blob[]>([])
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const streamRef = useRef<MediaStream | null>(null)

  useEffect(() => {
    return () => {
      stopTimer()
      stopStream()
    }
  }, [])

  function stopTimer() {
    if (timerRef.current) {
      clearInterval(timerRef.current)
      timerRef.current = null
    }
  }

  function stopStream() {
    streamRef.current?.getTracks().forEach(t => t.stop())
    streamRef.current = null
  }

  async function startRecording() {
    setErrorMsg('')
    setElapsed(0)
    chunksRef.current = []

    let stream: MediaStream
    try {
      stream = await navigator.mediaDevices.getUserMedia({ audio: true })
    } catch {
      setErrorMsg('Permissão de microfone negada. Habilite o acesso ao microfone nas configurações do navegador.')
      setState('error')
      return
    }

    streamRef.current = stream
    const mr = new MediaRecorder(stream, { mimeType: preferredMimeType() })
    mediaRecorderRef.current = mr

    mr.ondataavailable = e => {
      if (e.data.size > 0) chunksRef.current.push(e.data)
    }

    mr.onstop = () => {
      stopStream()
      stopTimer()
      const blob = new Blob(chunksRef.current, { type: mr.mimeType })
      onAudioReady(blob)
      setState('stopped')
    }

    mr.start(250)
    setState('recording')

    timerRef.current = setInterval(() => {
      setElapsed(prev => {
        const next = prev + 1
        if (next >= MAX_DURATION_S) {
          stopRecording()
        }
        return next
      })
    }, 1000)
  }

  function stopRecording() {
    stopTimer()
    if (mediaRecorderRef.current?.state === 'recording') {
      mediaRecorderRef.current.stop()
    }
  }

  function reset() {
    setState('idle')
    setElapsed(0)
    setErrorMsg('')
  }

  const isWarning = elapsed >= WARN_THRESHOLD_S
  const progressPct = Math.min((elapsed / MAX_DURATION_S) * 100, 100)

  return (
    <div className="flex flex-col items-center gap-4">
      {state === 'idle' && (
        <button
          onClick={startRecording}
          disabled={disabled}
          className="w-20 h-20 rounded-full bg-navy-700 text-white flex items-center justify-center shadow-lg
                     hover:bg-navy-600 active:bg-navy-800 disabled:opacity-50 disabled:cursor-not-allowed
                     transition-colors focus:outline-none focus:ring-4 focus:ring-navy-300"
          aria-label="Iniciar gravação"
        >
          <MicIcon />
        </button>
      )}

      {state === 'recording' && (
        <div className="flex flex-col items-center gap-3">
          <button
            onClick={stopRecording}
            className="w-20 h-20 rounded-full bg-red-600 text-white flex items-center justify-center shadow-lg
                       hover:bg-red-500 active:bg-red-700 transition-colors
                       focus:outline-none focus:ring-4 focus:ring-red-300 animate-pulse"
            aria-label="Parar gravação"
          >
            <StopIcon />
          </button>
          <div className="w-48">
            <div className="flex justify-between text-xs text-gray-500 mb-1">
              <span>{formatTime(elapsed)}</span>
              <span className={isWarning ? 'text-red-500 font-medium' : ''}>
                {isWarning ? `Limite em ${MAX_DURATION_S - elapsed}s` : `Máx. ${MAX_DURATION_S}s`}
              </span>
            </div>
            <div className="h-1.5 rounded-full bg-gray-200 overflow-hidden">
              <div
                className={`h-full rounded-full transition-all ${isWarning ? 'bg-red-500' : 'bg-navy-700'}`}
                style={{ width: `${progressPct}%` }}
                role="progressbar"
                aria-valuenow={elapsed}
                aria-valuemax={MAX_DURATION_S}
              />
            </div>
          </div>
          <p className="text-sm text-grafite-600 animate-pulse">Gravando...</p>
        </div>
      )}

      {state === 'stopped' && (
        <div className="flex flex-col items-center gap-2">
          <div className="w-20 h-20 rounded-full bg-gray-200 flex items-center justify-center">
            <CheckCircleIcon />
          </div>
          <p className="text-sm text-grafite-600">Áudio capturado ({formatTime(elapsed)})</p>
          <button
            onClick={reset}
            className="text-xs text-gray-500 underline hover:text-grafite-600"
          >
            Gravar novamente
          </button>
        </div>
      )}

      {state === 'error' && (
        <div className="flex flex-col items-center gap-3">
          <div className="w-20 h-20 rounded-full bg-gray-100 flex items-center justify-center">
            <MicOffIcon />
          </div>
          <p className="text-sm text-red-600 text-center max-w-xs">{errorMsg}</p>
          <button
            onClick={reset}
            className="text-xs text-gray-500 underline hover:text-grafite-600"
          >
            Tentar novamente
          </button>
        </div>
      )}
    </div>
  )
}

function formatTime(seconds: number): string {
  const m = Math.floor(seconds / 60)
  const s = seconds % 60
  return `${m}:${s.toString().padStart(2, '0')}`
}

function preferredMimeType(): string {
  const types = ['audio/webm;codecs=opus', 'audio/webm', 'audio/ogg;codecs=opus', 'audio/mp4']
  for (const t of types) {
    if (MediaRecorder.isTypeSupported(t)) return t
  }
  return ''
}

function MicIcon() {
  return (
    <svg width="32" height="32" viewBox="0 0 32 32" fill="none" aria-hidden="true">
      <rect x="11" y="4" width="10" height="16" rx="5" stroke="white" strokeWidth="2" />
      <path d="M6 18c0 5.523 4.477 10 10 10s10-4.477 10-10" stroke="white" strokeWidth="2" strokeLinecap="round" />
      <line x1="16" y1="28" x2="16" y2="32" stroke="white" strokeWidth="2" strokeLinecap="round" />
    </svg>
  )
}

function StopIcon() {
  return (
    <svg width="24" height="24" viewBox="0 0 24 24" fill="white" aria-hidden="true">
      <rect x="4" y="4" width="16" height="16" rx="2" />
    </svg>
  )
}

function CheckCircleIcon() {
  return (
    <svg width="40" height="40" viewBox="0 0 40 40" fill="none" aria-hidden="true">
      <circle cx="20" cy="20" r="18" stroke="#1e3a5f" strokeWidth="2" />
      <path d="M12 20l6 6 10-12" stroke="#1e3a5f" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  )
}

function MicOffIcon() {
  return (
    <svg width="40" height="40" viewBox="0 0 40 40" fill="none" aria-hidden="true">
      <line x1="8" y1="8" x2="32" y2="32" stroke="#9e9e9e" strokeWidth="2.5" strokeLinecap="round" />
      <rect x="14" y="6" width="12" height="20" rx="6" stroke="#9e9e9e" strokeWidth="2" />
    </svg>
  )
}
