// Live audio room adapter (Phase 4.2).
//
// Live media NEVER flows through the app's HTTP API. This
// module connects a browser participant to a LiveKit/WebRTC
// room using the short-lived participant tokens minted by the
// backend (`LiveAccessToken`). Provider credentials never
// reach this file — only the signed token and the public
// connection URL do.
//
// `livekit-client` is loaded through a runtime dynamic import
// so the app builds and serves even before the package is
// installed; connecting then fails with a clear message
// instead of a broken bundle.

// The minimal livekit-client surface this adapter relies on.
// Kept local (instead of importing the package types) so the
// project typechecks without the dependency installed.
interface LiveKitRemoteAudioTrack {
  kind: string
  attach(
    element?: HTMLAudioElement,
  ): HTMLAudioElement
  detach(): HTMLAudioElement[]
}

interface LiveKitRoomLike {
  state: string
  connect(
    url: string,
    token: string,
  ): Promise<void>
  disconnect(): void
  on(
    event: string,
    handler: (...args: unknown[]) => void,
  ): unknown
  removeAllListeners?(): void
  localParticipant: {
    setMicrophoneEnabled(
      enabled: boolean,
    ): Promise<void>
  }
}

type LiveKitModule = {
  Room: new (
    options?: Record<string, unknown>,
  ) => LiveKitRoomLike
  RoomEvent: Record<string, string>
}

async function loadLiveKit(): Promise<LiveKitModule> {
  try {
    // livekit-client is an installed frontend dependency.
    // Use a normal literal dynamic import so Vite resolves
    // and bundles the package correctly.
    const imported = await import('livekit-client')

    return imported as unknown as LiveKitModule
  } catch (err) {
    console.error('Failed to load livekit-client:', err)

    throw new Error(
      'Live audio could not be initialized. Please refresh the page and try again.',
    )
  }
}

export type LiveRoomState =
  | 'idle'
  | 'connecting'
  | 'connected'
  | 'reconnecting'
  | 'disconnected'
  | 'failed'

export type LiveRoomError = {
  message: string
}

type LiveRoomStateListener = (
  state: LiveRoomState,
) => void

// LiveAudioRoom owns ONE LiveKit room connection bound to a
// single <audio> element. The element is passed in by the
// caller so the rest of the app keeps full control of audio
// playback surfaces.
export class LiveAudioRoom {
  private audioElement: HTMLAudioElement
  private room: LiveKitRoomLike | null = null
  private listeners = new Set<
    LiveRoomStateListener
  >()
  private currentState: LiveRoomState = 'idle'

  constructor(
    audioElement: HTMLAudioElement,
  ) {
    this.audioElement = audioElement
  }

  getState(): LiveRoomState {
    return this.currentState
  }

  onStateChange(
    listener: LiveRoomStateListener,
  ): () => void {
    this.listeners.add(listener)

    return () => {
      this.listeners.delete(listener)
    }
  }

  // connect joins the room. `publish` decides whether the
  // local microphone is captured: hosts publish, listeners
  // subscribe only (the server-side token would reject a
  // listener's publish attempt anyway).
  async connect(
    connectURL: string,
    token: string,
    publish: boolean,
  ): Promise<void> {
    if (this.room) {
      throw new Error(
        'Live room is already connected',
      )
    }

    this.setState('connecting')

    try {
      const livekit =
        await loadLiveKit()

      const room =
        new livekit.Room({
          adaptiveStream: true,
          dynacast: true,
        })

      this.attachRoomEvents(
        livekit.RoomEvent,
        room,
      )

      await room.connect(
        connectURL,
        token,
      )

      this.room = room

      if (publish) {
        await room.localParticipant
          .setMicrophoneEnabled(
            true,
          )
      }

      this.setState('connected')
    } catch (err) {
      this.room = null

      this.setState('failed')

      throw err instanceof Error
        ? err
        : new Error(
            'Failed to connect to the live room',
          )
    }
  }

  // setMicrophone toggles the host's local microphone.
  async setMicrophone(
    enabled: boolean,
  ): Promise<void> {
    if (!this.room) {
      throw new Error(
        'Live room is not connected',
      )
    }

    await this.room.localParticipant
      .setMicrophoneEnabled(enabled)
  }

  disconnect(): void {
    if (!this.room) {
      return
    }

    try {
      this.room.disconnect()
    } finally {
      this.room = null

      this.detachAudio()

      this.setState('disconnected')
    }
  }

  private attachRoomEvents(
    roomEvent: Record<string, string>,
    room: LiveKitRoomLike,
  ): void {
    const subscribed =
      roomEvent.TrackSubscribed ??
      'trackSubscribed'

    room.on(
      subscribed,
      (...args: unknown[]) => {
        const track =
          args[0] as
            | LiveKitRemoteAudioTrack
            | undefined

        if (track?.kind !== 'audio') {
          return
        }

        track.attach(
          this.audioElement,
        )
      },
    )

    const unsubscribed =
      roomEvent.TrackUnsubscribed ??
      'trackUnsubscribed'

    room.on(
      unsubscribed,
      (...args: unknown[]) => {
        const track =
          args[0] as
            | LiveKitRemoteAudioTrack
            | undefined

        if (track?.kind !== 'audio') {
          return
        }

        try {
          track.detach()
        } catch {
          // Detaching an already-gone track is harmless.
        }
      },
    )

    room.on(
      roomEvent.Reconnecting ??
        'reconnecting',
      () => {
        this.setState('reconnecting')
      },
    )

    room.on(
      roomEvent.Reconnected ??
        'reconnected',
      () => {
        this.setState('connected')
      },
    )

    room.on(
      roomEvent.Disconnected ??
        'disconnected',
      () => {
        this.room = null

        this.detachAudio()

        this.setState('disconnected')
      },
    )
  }

  private detachAudio(): void {
    this.audioElement.srcObject = null

    this.audioElement.removeAttribute(
      'src',
    )
  }

  private setState(
    state: LiveRoomState,
  ): void {
    this.currentState = state

    for (const listener of this.listeners) {
      listener(state)
    }
  }
}
