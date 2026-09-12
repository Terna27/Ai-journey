import type {
  APIErrorResponse,
  Artist,
  LoginRequest,
  LoginResponse,
  MeResponse,
  RegisterRequest,
  RegisterResponse,
  ResendVerificationRequest,
  ResendVerificationResponse,
  VerifyEmailRequest,
  VerifyEmailResponse,
} from '../types/auth'

import type {
  SearchOptions,
  SearchResponse,
} from '../types/search'

import type {
  DiscoveryHeroResponse,
  DiscoveryResponse,
} from '../types/discovery'

import type {
  Music,
  UploadMusicInput,
} from '../types/music'

import type {
  CreatePlaylistInput,
  Playlist,
  PlaylistDetails,
  UpdatePlaylistInput,
} from '../types/playlist'

import type {
  ArtistProfileResponse,
} from '../types/artist'

import type {
  AddReleaseTrackInput,
  CreateReleaseInput,
  Release,
  ReleaseDetails,
  UpdateReleaseInput,
} from '../types/release'

import type {
  CreatePlaybackSessionInput,
  ListeningHistoryItem,
  PlaybackSession,
  UpdatePlaybackProgressInput,
} from '../types/playback'

import type {
  CreatePodcastEpisodeInput,
  CreatePodcastInput,
  Podcast,
  PodcastDetails,
  PodcastEpisode,
  UpdatePodcastEpisodeInput,
  UpdatePodcastEpisodeMediaInput,
  UpdatePodcastArtworkInput,
  UpdatePodcastInput,
} from '../types/podcast'

import type {
  CreatePodcastPlaybackSessionInput,
  PodcastContinueListeningItem,
  PodcastListeningHistoryItem,
  PodcastPlaybackSession,
  UpdatePodcastPlaybackProgressInput,
} from '../types/podcastPlayback'

import type {
  LiveAccessToken,
  PodcastLiveBroadcast,
  PodcastLiveDetails,
} from '../types/podcastLive'

const API_BASE_URL =
  import.meta.env.VITE_API_BASE_URL ??
  'http://localhost:8080/api/v1'

export class APIError extends Error {
  readonly status: number
  readonly code: string

  constructor(
    message: string,
    status: number,
    code: string,
  ) {
    super(message)

    this.name = 'APIError'
    this.status = status
    this.code = code
  }
}

async function request<T>(
  path: string,
  options: RequestInit = {},
): Promise<T> {
  const response = await fetch(
    `${API_BASE_URL}${path}`,
    {
      ...options,
      headers: {
        ...options.headers,
      },
    },
  )

  if (!response.ok) {
    let message = 'Something went wrong'
    let code = ''

    try {
      const body =
        (await response.json()) as APIErrorResponse

      if (body.error?.message) {
        message = body.error.message
      }

      if (body.error?.code) {
        code = body.error.code
      }
    } catch {
      // Response did not contain JSON.
    }

    throw new APIError(
      message,
      response.status,
      code,
    )
  }

  if (response.status === 204) {
    return undefined as T
  }

  return response.json() as Promise<T>
}

export async function registerUser(
  payload: RegisterRequest,
): Promise<RegisterResponse> {
  return request<RegisterResponse>(
    '/auth/register',
    {
      method: 'POST',
      headers: {
        'Content-Type':
          'application/json',
      },
      body: JSON.stringify(payload),
    },
  )
}

export async function loginUser(
  payload: LoginRequest,
): Promise<LoginResponse> {
  return request<LoginResponse>(
    '/auth/login',
    {
      method: 'POST',
      headers: {
        'Content-Type':
          'application/json',
      },
      body: JSON.stringify(payload),
    },
  )
}

export async function verifyEmail(
  payload: VerifyEmailRequest,
): Promise<VerifyEmailResponse> {
  return request<VerifyEmailResponse>(
    '/auth/verify-email',
    {
      method: 'POST',
      headers: {
        'Content-Type':
          'application/json',
      },
      body: JSON.stringify(payload),
    },
  )
}

export async function resendVerification(
  payload: ResendVerificationRequest,
): Promise<ResendVerificationResponse> {
  return request<ResendVerificationResponse>(
    '/auth/resend-verification',
    {
      method: 'POST',
      headers: {
        'Content-Type':
          'application/json',
      },
      body: JSON.stringify(payload),
    },
  )
}

export async function getMe(
  token: string,
): Promise<MeResponse> {
  return request<MeResponse>(
    '/me',
    {
      headers: {
        Authorization:
          `Bearer ${token}`,
      },
    },
  )
}

export async function createArtistProfile(
  token: string,
): Promise<Artist> {
  return request<Artist>(
    '/artists/profile',
    {
      method: 'POST',
      headers: {
        Authorization:
          `Bearer ${token}`,
      },
    },
  )
}

export async function getMusic():
Promise<Music[]> {
  return request<Music[]>(
    '/music',
  )
}

export async function uploadMusic(
  payload: UploadMusicInput,
  token: string,
): Promise<Music> {
  const formData = new FormData()

  formData.append(
    'artist_name',
    payload.artistName,
  )

  formData.append(
    'song_title',
    payload.songTitle,
  )

  formData.append(
    'genre',
    payload.genre,
  )

  formData.append(
    'audio_key',
    `${Date.now()}-${payload.audio.name}`,
  )

  formData.append(
    'image',
    payload.image,
  )

  formData.append(
    'audio',
    payload.audio,
  )

  return request<Music>(
    '/music',
    {
      method: 'POST',
      headers: {
        Authorization:
          `Bearer ${token}`,
      },
      body: formData,
    },
  )
}

export async function getLikedMusic(
  token: string,
): Promise<Music[]> {
  return request<Music[]>(
    '/me/liked-music',
    {
      headers: {
        Authorization:
          `Bearer ${token}`,
      },
    },
  )
}

export async function likeMusic(
  musicID: number,
  token: string,
): Promise<Music> {
  return request<Music>(
    `/music/${musicID}/like`,
    {
      method: 'POST',
      headers: {
        Authorization:
          `Bearer ${token}`,
      },
    },
  )
}

export async function unlikeMusic(
  musicID: number,
  token: string,
): Promise<Music> {
  return request<Music>(
    `/music/${musicID}/like`,
    {
      method: 'DELETE',
      headers: {
        Authorization:
          `Bearer ${token}`,
      },
    },
  )
}

export async function createPlaylist(
  payload: CreatePlaylistInput,
  token: string,
): Promise<Playlist> {
  return request<Playlist>(
    '/playlists',
    {
      method: 'POST',
      headers: {
        Authorization:
          `Bearer ${token}`,
        'Content-Type':
          'application/json',
      },
      body: JSON.stringify(payload),
    },
  )
}

export async function getMyPlaylists(
  token: string,
): Promise<Playlist[]> {
  return request<Playlist[]>(
    '/me/playlists',
    {
      headers: {
        Authorization:
          `Bearer ${token}`,
      },
    },
  )
}

export async function getPlaylist(
  playlistID: number,
  token: string,
): Promise<PlaylistDetails> {
  return request<PlaylistDetails>(
    `/playlists/${playlistID}`,
    {
      headers: {
        Authorization:
          `Bearer ${token}`,
      },
    },
  )
}

export async function updatePlaylist(
  playlistID: number,
  payload: UpdatePlaylistInput,
  token: string,
): Promise<Playlist> {
  return request<Playlist>(
    `/playlists/${playlistID}`,
    {
      method: 'PUT',
      headers: {
        Authorization:
          `Bearer ${token}`,
        'Content-Type':
          'application/json',
      },
      body: JSON.stringify(payload),
    },
  )
}

export async function deletePlaylist(
  playlistID: number,
  token: string,
): Promise<void> {
  await request<void>(
    `/playlists/${playlistID}`,
    {
      method: 'DELETE',
      headers: {
        Authorization:
          `Bearer ${token}`,
      },
    },
  )
}

export async function addTrackToPlaylist(
  playlistID: number,
  musicID: number,
  token: string,
): Promise<{ message: string }> {
  return request<{ message: string }>(
    `/playlists/${playlistID}/tracks/${musicID}`,
    {
      method: 'POST',
      headers: {
        Authorization:
          `Bearer ${token}`,
      },
    },
  )
}

export async function removeTrackFromPlaylist(
  playlistID: number,
  musicID: number,
  token: string,
): Promise<void> {
  await request<void>(
    `/playlists/${playlistID}/tracks/${musicID}`,
    {
      method: 'DELETE',
      headers: {
        Authorization:
          `Bearer ${token}`,
      },
    },
  )
}

export async function getArtistProfile(
  artistID: number,
): Promise<ArtistProfileResponse> {
  return request<ArtistProfileResponse>(
    `/artists/${artistID}`,
  )
}

export async function getArtistReleases(
  artistID: number,
): Promise<Release[]> {
  return request<Release[]>(
    `/artists/${artistID}/releases`,
  )
}

export async function getRelease(
  releaseID: number,
): Promise<ReleaseDetails> {
  return request<ReleaseDetails>(
    `/releases/${releaseID}`,
  )
}

export async function getMyReleases(
  token: string,
): Promise<Release[]> {
  return request<Release[]>(
    '/me/releases',
    {
      headers: {
        Authorization:
          `Bearer ${token}`,
      },
    },
  )
}

export async function getMyRelease(
  releaseID: number,
  token: string,
): Promise<ReleaseDetails> {
  return request<ReleaseDetails>(
    `/me/releases/${releaseID}`,
    {
      headers: {
        Authorization:
          `Bearer ${token}`,
      },
    },
  )
}

export async function createRelease(
  payload: CreateReleaseInput,
  token: string,
): Promise<Release> {
  return request<Release>(
    '/releases',
    {
      method: 'POST',
      headers: {
        Authorization:
          `Bearer ${token}`,
        'Content-Type':
          'application/json',
      },
      body: JSON.stringify(payload),
    },
  )
}

export async function updateRelease(
  releaseID: number,
  payload: UpdateReleaseInput,
  token: string,
): Promise<Release> {
  return request<Release>(
    `/releases/${releaseID}`,
    {
      method: 'PUT',
      headers: {
        Authorization:
          `Bearer ${token}`,
        'Content-Type':
          'application/json',
      },
      body: JSON.stringify(payload),
    },
  )
}

export async function deleteRelease(
  releaseID: number,
  token: string,
): Promise<void> {
  await request<void>(
    `/releases/${releaseID}`,
    {
      method: 'DELETE',
      headers: {
        Authorization:
          `Bearer ${token}`,
      },
    },
  )
}

export async function addTrackToRelease(
  releaseID: number,
  musicID: number,
  payload: AddReleaseTrackInput,
  token: string,
): Promise<ReleaseDetails> {
  return request<ReleaseDetails>(
    `/releases/${releaseID}/tracks/${musicID}`,
    {
      method: 'POST',
      headers: {
        Authorization:
          `Bearer ${token}`,
        'Content-Type':
          'application/json',
      },
      body: JSON.stringify(payload),
    },
  )
}

export async function removeTrackFromRelease(
  releaseID: number,
  musicID: number,
  token: string,
): Promise<void> {
  await request<void>(
    `/releases/${releaseID}/tracks/${musicID}`,
    {
      method: 'DELETE',
      headers: {
        Authorization:
          `Bearer ${token}`,
      },
    },
  )
}


export async function searchMusic(
  options: SearchOptions,
): Promise<SearchResponse> {
  const params = new URLSearchParams()

  params.set(
    'q',
    options.query,
  )

  if (options.type) {
    params.set(
      'type',
      options.type,
    )
  }

  if (options.sort) {
    params.set(
      'sort',
      options.sort,
    )
  }

  if (options.genre?.trim()) {
    params.set(
      'genre',
      options.genre.trim(),
    )
  }

  if (options.page) {
    params.set(
      'page',
      String(options.page),
    )
  }

  if (options.limit) {
    params.set(
      'limit',
      String(options.limit),
    )
  }

  return request<SearchResponse>(
    `/search?${params.toString()}`,
  )
}


export async function getDiscovery():
Promise<DiscoveryResponse> {
  return request<DiscoveryResponse>(
    '/discover',
  )
}

export async function getHeroArtists():
Promise<DiscoveryHeroResponse> {
  return request<DiscoveryHeroResponse>(
    '/discovery/hero-artists',
  )
}


export async function updateArtistProfile(
  formData: FormData,
  token: string,
): Promise<
  import('../types/auth').Artist
> {
  return request<
    import('../types/auth').Artist
  >(
    '/me/artist-profile',
    {
      method: 'PATCH',
      headers: {
        Authorization:
          `Bearer ${token}`,
      },
      body: formData,
    },
  )
}

export { API_BASE_URL }

export async function getTrackLyrics(
  musicID: number,
): Promise<
  import('../types/lyrics').TrackLyrics
> {
  return request<
    import('../types/lyrics').TrackLyrics
  >(
    `/music/${musicID}/lyrics`,
  )
}


export type SaveTrackLyricsInput = {
  plain_lyrics: string
  synced_lines: import('../types/lyrics').LyricLine[]
}
export async function saveTrackLyrics(
  musicID: number,
  payload: SaveTrackLyricsInput,
  token: string,
): Promise<
  import('../types/lyrics').TrackLyrics
> {
  return request<
    import('../types/lyrics').TrackLyrics
  >(
    `/music/${musicID}/lyrics`,
    {
      method: 'PUT',
      headers: {
        Authorization:
          `Bearer ${token}`,
        'Content-Type':
          'application/json',
      },
      body: JSON.stringify(payload),
    },
  )
}

export async function deleteTrackLyrics(
  musicID: number,
  token: string,
): Promise<void> {
  await request<void>(
    `/music/${musicID}/lyrics`,
    {
      method: 'DELETE',
      headers: {
        Authorization:
          `Bearer ${token}`,
      },
    },
  )
}

// -----------------------------------------------------------------
// Playback tracking (Phase 3.8)
// -----------------------------------------------------------------

export async function createPlaybackSession(
  payload: CreatePlaybackSessionInput,
  token: string,
): Promise<PlaybackSession> {
  return request<PlaybackSession>(
    '/playback/sessions',
    {
      method: 'POST',
      headers: {
        Authorization:
          `Bearer ${token}`,
        'Content-Type':
          'application/json',
      },
      body: JSON.stringify(
        payload,
      ),
    },
  )
}

export async function getPlaybackSession(
  sessionID: string,
  token: string,
): Promise<PlaybackSession> {
  return request<PlaybackSession>(
    `/playback/sessions/${sessionID}`,
    {
      headers: {
        Authorization:
          `Bearer ${token}`,
      },
    },
  )
}

export async function updatePlaybackProgress(
  sessionID: string,
  payload: UpdatePlaybackProgressInput,
  token: string,
): Promise<PlaybackSession> {
  return request<PlaybackSession>(
    `/playback/sessions/${sessionID}`,
    {
      method: 'PATCH',
      headers: {
        Authorization:
          `Bearer ${token}`,
        'Content-Type':
          'application/json',
      },
      body: JSON.stringify(
        payload,
      ),
    },
  )
}

export async function completePlaybackSession(
  sessionID: string,
  payload: UpdatePlaybackProgressInput,
  token: string,
  keepalive = false,
): Promise<PlaybackSession> {
  return request<PlaybackSession>(
    `/playback/sessions/${sessionID}/complete`,
    {
      method: 'POST',
      keepalive,
      headers: {
        Authorization:
          `Bearer ${token}`,
        'Content-Type':
          'application/json',
      },
      body: JSON.stringify(
        payload,
      ),
    },
  )
}

export async function getListeningHistory(
  token: string,
  limit = 20,
  offset = 0,
): Promise<
  ListeningHistoryItem[]
> {
  return request<
    ListeningHistoryItem[]
  >(
    `/me/listening-history?limit=${limit}&offset=${offset}`,
    {
      headers: {
        Authorization:
          `Bearer ${token}`,
      },
    },
  )
}

// -----------------------------------------------------------------
// Podcast playback tracking (Phase 4.1)
//
// Dedicated podcast endpoints. Podcast episode ids must
// never be sent to the music playback session functions
// above.
// -----------------------------------------------------------------

export async function createPodcastPlaybackSession(
  payload: CreatePodcastPlaybackSessionInput,
  token: string,
): Promise<PodcastPlaybackSession> {
  return request<PodcastPlaybackSession>(
    '/podcast-playback/sessions',
    {
      method: 'POST',
      headers: {
        Authorization:
          `Bearer ${token}`,
        'Content-Type':
          'application/json',
      },
      body: JSON.stringify(
        payload,
      ),
    },
  )
}

export async function getPodcastPlaybackSession(
  sessionID: string,
  token: string,
): Promise<PodcastPlaybackSession> {
  return request<PodcastPlaybackSession>(
    `/podcast-playback/sessions/${sessionID}`,
    {
      headers: {
        Authorization:
          `Bearer ${token}`,
      },
    },
  )
}

export async function updatePodcastPlaybackProgress(
  sessionID: string,
  payload: UpdatePodcastPlaybackProgressInput,
  token: string,
): Promise<PodcastPlaybackSession> {
  return request<PodcastPlaybackSession>(
    `/podcast-playback/sessions/${sessionID}`,
    {
      method: 'PATCH',
      headers: {
        Authorization:
          `Bearer ${token}`,
        'Content-Type':
          'application/json',
      },
      body: JSON.stringify(
        payload,
      ),
    },
  )
}

export async function completePodcastPlaybackSession(
  sessionID: string,
  payload: UpdatePodcastPlaybackProgressInput,
  token: string,
  keepalive = false,
): Promise<PodcastPlaybackSession> {
  return request<PodcastPlaybackSession>(
    `/podcast-playback/sessions/${sessionID}/complete`,
    {
      method: 'POST',
      keepalive,
      headers: {
        Authorization:
          `Bearer ${token}`,
        'Content-Type':
          'application/json',
      },
      body: JSON.stringify(
        payload,
      ),
    },
  )
}

export async function getPodcastListeningHistory(
  token: string,
  limit = 20,
  offset = 0,
): Promise<
  PodcastListeningHistoryItem[]
> {
  return request<
    PodcastListeningHistoryItem[]
  >(
    `/me/podcast-listening-history?limit=${limit}&offset=${offset}`,
    {
      headers: {
        Authorization:
          `Bearer ${token}`,
      },
    },
  )
}

export async function getPodcastContinueListening(
  token: string,
  limit = 20,
  offset = 0,
): Promise<
  PodcastContinueListeningItem[]
> {
  return request<
    PodcastContinueListeningItem[]
  >(
    `/me/podcast-continue-listening?limit=${limit}&offset=${offset}`,
    {
      headers: {
        Authorization:
          `Bearer ${token}`,
      },
    },
  )
}

// -----------------------------------------------------------------
// Artist following (Phase 3.9)
// -----------------------------------------------------------------

export async function getArtistFollowerCount(
  artistID: number,
): Promise<
  import('../types/follow').ArtistFollowerCount
> {
  return request<
    import('../types/follow').ArtistFollowerCount
  >(
    `/artists/${artistID}/followers/count`,
  )
}

export async function getArtistFollowStatus(
  artistID: number,
  token: string,
): Promise<
  import('../types/follow').ArtistFollowStatus
> {
  return request<
    import('../types/follow').ArtistFollowStatus
  >(
    `/artists/${artistID}/follow`,
    {
      headers: {
        Authorization:
          `Bearer ${token}`,
      },
    },
  )
}

export async function followArtist(
  artistID: number,
  token: string,
): Promise<
  import('../types/follow').ArtistFollowStatus
> {
  return request<
    import('../types/follow').ArtistFollowStatus
  >(
    `/artists/${artistID}/follow`,
    {
      method: 'POST',
      headers: {
        Authorization:
          `Bearer ${token}`,
      },
    },
  )
}

export async function unfollowArtist(
  artistID: number,
  token: string,
): Promise<
  import('../types/follow').ArtistFollowStatus
> {
  return request<
    import('../types/follow').ArtistFollowStatus
  >(
    `/artists/${artistID}/follow`,
    {
      method: 'DELETE',
      headers: {
        Authorization:
          `Bearer ${token}`,
      },
    },
  )
}

// -----------------------------------------------------------------
// Podcasts (Phase 4.0)
// -----------------------------------------------------------------

export async function getPublishedPodcasts(
  limit = 20,
  offset = 0,
): Promise<Podcast[]> {
  return request<Podcast[]>(
    `/podcasts?limit=${limit}&offset=${offset}`,
  )
}

export async function getPublicPodcast(
  podcastID: number,
): Promise<PodcastDetails> {
  return request<PodcastDetails>(
    `/podcasts/${podcastID}`,
  )
}

export async function getPublicPodcastBySlug(
  slug: string,
): Promise<PodcastDetails> {
  return request<PodcastDetails>(
    `/podcasts/slug/${encodeURIComponent(slug)}`,
  )
}

export async function getPublicPodcastEpisode(
  episodeID: number,
): Promise<PodcastEpisode> {
  return request<PodcastEpisode>(
    `/podcast-episodes/${episodeID}`,
  )
}

export async function getMyPodcasts(
  token: string,
): Promise<Podcast[]> {
  return request<Podcast[]>(
    '/me/podcasts',
    {
      headers: {
        Authorization:
          `Bearer ${token}`,
      },
    },
  )
}

export async function getMyPodcast(
  podcastID: number,
  token: string,
): Promise<PodcastDetails> {
  return request<PodcastDetails>(
    `/me/podcasts/${podcastID}`,
    {
      headers: {
        Authorization:
          `Bearer ${token}`,
      },
    },
  )
}

export async function createPodcast(
  payload: CreatePodcastInput,
  token: string,
): Promise<Podcast> {
  return request<Podcast>(
    '/podcasts',
    {
      method: 'POST',
      headers: {
        Authorization:
          `Bearer ${token}`,
        'Content-Type':
          'application/json',
      },
      body: JSON.stringify(payload),
    },
  )
}

export async function updatePodcast(
  podcastID: number,
  payload: UpdatePodcastInput,
  token: string,
): Promise<Podcast> {
  return request<Podcast>(
    `/me/podcasts/${podcastID}`,
    {
      method: 'PATCH',
      headers: {
        Authorization:
          `Bearer ${token}`,
        'Content-Type':
          'application/json',
      },
      body: JSON.stringify(payload),
    },
  )
}

export async function updatePodcastArtwork(
  podcastID: number,
  payload: UpdatePodcastArtworkInput,
  token: string,
): Promise<Podcast> {
  const formData = new FormData()

  formData.append(
    'artwork',
    payload.artwork,
  )

  return request<Podcast>(
    `/me/podcasts/${podcastID}/artwork`,
    {
      method: 'PATCH',
      headers: {
        Authorization:
          `Bearer ${token}`,
      },
      body: formData,
    },
  )
}

export async function deletePodcast(
  podcastID: number,
  token: string,
): Promise<void> {
  return request<void>(
    `/me/podcasts/${podcastID}`,
    {
      method: 'DELETE',
      headers: {
        Authorization:
          `Bearer ${token}`,
      },
    },
  )
}

export async function createPodcastEpisode(
  podcastID: number,
  payload: CreatePodcastEpisodeInput,
  token: string,
): Promise<PodcastEpisode> {
  return request<PodcastEpisode>(
    `/me/podcasts/${podcastID}/episodes`,
    {
      method: 'POST',
      headers: {
        Authorization:
          `Bearer ${token}`,
        'Content-Type':
          'application/json',
      },
      body: JSON.stringify(payload),
    },
  )
}

export async function updatePodcastEpisode(
  episodeID: number,
  payload: UpdatePodcastEpisodeInput,
  token: string,
): Promise<PodcastEpisode> {
  return request<PodcastEpisode>(
    `/me/podcast-episodes/${episodeID}`,
    {
      method: 'PATCH',
      headers: {
        Authorization:
          `Bearer ${token}`,
        'Content-Type':
          'application/json',
      },
      body: JSON.stringify(payload),
    },
  )
}

export async function updatePodcastEpisodeMedia(
  episodeID: number,
  payload: UpdatePodcastEpisodeMediaInput,
  token: string,
): Promise<PodcastEpisode> {
  const formData = new FormData()

  if (payload.audio) {
    formData.append(
      'audio',
      payload.audio,
    )
  }

  if (payload.artwork) {
    formData.append(
      'artwork',
      payload.artwork,
    )
  }

  return request<PodcastEpisode>(
    `/me/podcast-episodes/${episodeID}/media`,
    {
      method: 'PATCH',
      headers: {
        Authorization:
          `Bearer ${token}`,
      },
      body: formData,
    },
  )
}

export async function deletePodcastEpisode(
  episodeID: number,
  token: string,
): Promise<void> {
  return request<void>(
    `/me/podcast-episodes/${episodeID}`,
    {
      method: 'DELETE',
      headers: {
        Authorization:
          `Bearer ${token}`,
      },
    },
  )
}

// -------------------------
// Podcast live sessions
// -------------------------

export async function schedulePodcastLiveEpisode(
  episodeID: number,
  scheduledStartAt: string,
  token: string,
): Promise<PodcastLiveDetails> {
  return request<PodcastLiveDetails>(
    `/me/podcast-episodes/${episodeID}/live/schedule`,
    {
      method: 'POST',
      headers: {
        Authorization:
          `Bearer ${token}`,
        'Content-Type':
          'application/json',
      },
      body: JSON.stringify({
        scheduled_start_at:
          scheduledStartAt,
      }),
    },
  )
}

export async function getOwnedPodcastLive(
  episodeID: number,
  token: string,
): Promise<PodcastLiveDetails> {
  return request<PodcastLiveDetails>(
    `/me/podcast-episodes/${episodeID}/live`,
    {
      method: 'GET',
      headers: {
        Authorization:
          `Bearer ${token}`,
      },
    },
  )
}

export async function startPodcastLive(
  episodeID: number,
  token: string,
): Promise<PodcastLiveDetails> {
  return request<PodcastLiveDetails>(
    `/me/podcast-episodes/${episodeID}/live/start`,
    {
      method: 'POST',
      headers: {
        Authorization:
          `Bearer ${token}`,
      },
    },
  )
}

export async function endPodcastLive(
  episodeID: number,
  token: string,
): Promise<PodcastLiveDetails> {
  return request<PodcastLiveDetails>(
    `/me/podcast-episodes/${episodeID}/live/end`,
    {
      method: 'POST',
      headers: {
        Authorization:
          `Bearer ${token}`,
      },
    },
  )
}

export async function cancelPodcastLive(
  episodeID: number,
  token: string,
): Promise<PodcastLiveDetails> {
  return request<PodcastLiveDetails>(
    `/me/podcast-episodes/${episodeID}/live/cancel`,
    {
      method: 'POST',
      headers: {
        Authorization:
          `Bearer ${token}`,
      },
    },
  )
}

// Publishes the finished recording of an ENDED live session
// as the episode's regular audio (ENDED -> PUBLISHED).
export async function publishPodcastLiveRecording(
  sessionID: string,
  token: string,
): Promise<PodcastLiveDetails> {
  return request<PodcastLiveDetails>(
    `/me/podcast-live-sessions/${sessionID}/publish-recording`,
    {
      method: 'POST',
      headers: {
        Authorization:
          `Bearer ${token}`,
      },
    },
  )
}

export async function getPodcastLiveHostToken(
  sessionID: string,
  token: string,
): Promise<LiveAccessToken> {
  return request<LiveAccessToken>(
    `/podcast-live-sessions/${sessionID}/host-token`,
    {
      method: 'POST',
      headers: {
        Authorization:
          `Bearer ${token}`,
      },
    },
  )
}

// Anonymous listening: token is optional. When present it
// only stabilizes the listener identity server-side.
export async function getPodcastLiveListenerToken(
  sessionID: string,
  token?: string,
): Promise<LiveAccessToken> {
  const headers: Record<
    string,
    string
  > = {}

  if (token) {
    headers.Authorization =
      `Bearer ${token}`
  }

  return request<LiveAccessToken>(
    `/podcast-live-sessions/${sessionID}/listener-token`,
    {
      method: 'POST',
      headers,
    },
  )
}

export async function getUpcomingPodcastLive(
  limit = 20,
  offset = 0,
): Promise<PodcastLiveBroadcast[]> {
  return request<PodcastLiveBroadcast[]>(
    `/podcast-live/upcoming?limit=${limit}&offset=${offset}`,
  )
}

export async function getCurrentPodcastLive(
  limit = 20,
  offset = 0,
): Promise<PodcastLiveBroadcast[]> {
  return request<PodcastLiveBroadcast[]>(
    `/podcast-live/current?limit=${limit}&offset=${offset}`,
  )
}

export async function getPodcastLive(
  sessionID: string,
): Promise<PodcastLiveBroadcast> {
  return request<PodcastLiveBroadcast>(
    `/podcast-live/${sessionID}`,
  )
}


