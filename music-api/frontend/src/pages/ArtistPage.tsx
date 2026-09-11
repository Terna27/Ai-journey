import {
  useEffect,
  useMemo,
  useState,
} from 'react'

import {
  Link,
  useNavigate,
  useParams,
} from 'react-router-dom'

import AddToPlaylistButton from '../components/music/AddToPlaylistButton'
import AddToQueueButton from '../components/music/AddToQueueButton'

import { useAuth } from '../context/AuthContext'
import { useLibrary } from '../context/LibraryContext'
import { usePlayer } from '../context/PlayerContext'

import {
  APIError,
  followArtist,
  getArtistFollowerCount,
  getArtistFollowStatus,
  getArtistProfile,
  getArtistReleases,
  unfollowArtist,
} from '../lib/api'

import type {
  PublicArtist,
} from '../types/artist'

import type { Music } from '../types/music'

import type {
  Release,
} from '../types/release'

function formatReleaseDate(
  value: string | null,
) {
  if (!value) {
    return ''
  }

  const date = new Date(value)

  if (Number.isNaN(date.getTime())) {
    return ''
  }

  return new Intl.DateTimeFormat(
    'en',
    {
      day: 'numeric',
      month: 'short',
      year: 'numeric',
    },
  ).format(date)
}

function ArtistPage() {
  const { id } = useParams()

  const navigate = useNavigate()

  const {
    artist: authenticatedArtist,
    token,
    isAuthenticated,
    isLoadingIdentity,
  } = useAuth()

  const {
    currentTrack,
    isPlaying,
    playTrack,
  } = usePlayer()

  const {
    isLiked,
    toggleLike,
  } = useLibrary()

  const [artist, setArtist] =
    useState<PublicArtist | null>(null)

  const [tracks, setTracks] =
    useState<Music[]>([])

  const [releases, setReleases] =
    useState<Release[]>([])

  const [isLoading, setIsLoading] =
    useState(true)

  const [error, setError] =
    useState('')

  const [notFound, setNotFound] =
    useState(false)

  const [
    pendingLikeID,
    setPendingLikeID,
  ] = useState<number | null>(null)

  const [
    heroVideoFailed,
    setHeroVideoFailed,
  ] = useState(false)


  const [
    isFollowing,
    setIsFollowing,
  ] = useState(false)

  const [
    followerCount,
    setFollowerCount,
  ] = useState(0)

  const [
    isFollowLoading,
    setIsFollowLoading,
  ] = useState(false)

  const [
    followError,
    setFollowError,
  ] = useState('')

  const [
    shareMessage,
    setShareMessage,
  ] = useState('')

  const artistID = Number(id)

  const isValidArtistID =
    Number.isInteger(artistID) &&
    artistID > 0

  useEffect(() => {
    if (!isValidArtistID) {
      setIsLoading(false)
      return
    }

    let cancelled = false

    async function loadArtist() {
      try {
        setIsLoading(true)
        setError('')
        setNotFound(false)

        const [
          profileResponse,
          releaseResponse,
        ] = await Promise.all([
          getArtistProfile(
            artistID,
          ),
          getArtistReleases(
            artistID,
          ),
        ])

        if (cancelled) {
          return
        }

        setArtist(
          profileResponse.artist,
        )

        setTracks(
          profileResponse.tracks,
        )

        setReleases(
          releaseResponse,
        )
      } catch (err) {
        if (cancelled) {
          return
        }

        if (
          err instanceof APIError &&
          err.status === 404
        ) {
          setNotFound(true)
          return
        }

        setError(
          err instanceof Error
            ? err.message
            : 'Failed to load artist',
        )
      } finally {
        if (!cancelled) {
          setIsLoading(false)
        }
      }
    }

    void loadArtist()

    return () => {
      cancelled = true
    }
  }, [
    artistID,
    isValidArtistID,
  ])

  useEffect(() => {
    setHeroVideoFailed(false)
  }, [artist?.hero_video_url])


  useEffect(() => {
    if (!isValidArtistID) {
      return
    }

    let cancelled = false

    async function loadFollowState() {
      try {
        setFollowError('')

        const countResponse =
          await getArtistFollowerCount(
            artistID,
          )

        if (cancelled) {
          return
        }

        setFollowerCount(
          countResponse.follower_count,
        )

        if (
          !token ||
          !isAuthenticated
        ) {
          setIsFollowing(false)
          return
        }

        const status =
          await getArtistFollowStatus(
            artistID,
            token,
          )

        if (cancelled) {
          return
        }

        setIsFollowing(
          status.is_following,
        )

        setFollowerCount(
          status.follower_count,
        )
      } catch (err) {
        if (cancelled) {
          return
        }

        setFollowError(
          err instanceof Error
            ? err.message
            : 'Unable to load follow status',
        )
      }
    }

    void loadFollowState()

    return () => {
      cancelled = true
    }
  }, [
    artistID,
    isAuthenticated,
    isValidArtistID,
    token,
  ])

  const standaloneTracks =
    useMemo(
      () =>
        tracks.filter(
          (track) =>
            track.release_id === null,
        ),
      [tracks],
    )

  const playableTracks =
    useMemo(
      () =>
        tracks.filter(
          (track) =>
            Boolean(track.audio_url),
        ),
      [tracks],
    )

  function handlePlay(
    track: Music,
  ) {
    if (!isAuthenticated) {
      navigate(
        '/login',
        {
          state: {
            message:
              'Please log in to play music.',
            from:
              `/artists/${artistID}`,
          },
        },
      )

      return
    }

    playTrack(
      track,
      standaloneTracks,
    )
  }

  function handleHeroPlay() {
    if (
      playableTracks.length === 0
    ) {
      return
    }

    if (!isAuthenticated) {
      navigate(
        '/login',
        {
          state: {
            message:
              'Please log in to play music.',
            from:
              `/artists/${artistID}`,
          },
        },
      )

      return
    }

    playTrack(
      playableTracks[0],
      playableTracks,
    )
  }

  async function handleLike(
    track: Music,
  ) {
    if (!isAuthenticated) {
      navigate(
        '/login',
        {
          state: {
            message:
              'Please log in to like music.',
            from:
              `/artists/${artistID}`,
          },
        },
      )

      return
    }

    if (pendingLikeID !== null) {
      return
    }

    try {
      setPendingLikeID(track.id)

      await toggleLike(track)
    } catch {
      // LibraryContext exposes
      // the library error state.
    } finally {
      setPendingLikeID(null)
    }
  }

  async function handleFollowToggle() {
    if (!isAuthenticated || !token) {
      navigate(
        '/login',
        {
          state: {
            message:
              'Please log in to follow artists.',
            from:
              `/artists/${artistID}`,
          },
        },
      )

      return
    }

    if (isFollowLoading) {
      return
    }

    try {
      setIsFollowLoading(true)
      setFollowError('')

      const status =
        isFollowing
          ? await unfollowArtist(
              artistID,
              token,
            )
          : await followArtist(
              artistID,
              token,
            )

      setIsFollowing(
        status.is_following,
      )

      setFollowerCount(
        status.follower_count,
      )
    } catch (err) {
      setFollowError(
        err instanceof Error
          ? err.message
          : 'Unable to update follow status',
      )
    } finally {
      setIsFollowLoading(false)
    }
  }

  async function handleShare() {
    if (!artist) {
      return
    }

    const shareData = {
      title: artist.name,
      text:
        `Listen to ${artist.name}`,
      url: window.location.href,
    }

    try {
      if (navigator.share) {
        await navigator.share(
          shareData,
        )

        setShareMessage(
          'Artist shared',
        )

        return
      }

      await navigator.clipboard.writeText(
        window.location.href,
      )

      setShareMessage(
        'Artist link copied',
      )
    } catch (err) {
      if (
        err instanceof DOMException &&
        err.name === 'AbortError'
      ) {
        return
      }

      setShareMessage(
        'Unable to share artist',
      )
    }
  }

  if (!isValidArtistID) {
    return (
      <section className="content-panel">
        <h3>
          Artist not found
        </h3>

        <p>
          That artist link does not
          look right.{' '}
          <Link to="/">
            Back to music
          </Link>
          .
        </p>
      </section>
    )
  }

  if (isLoading) {
    return (
      <section className="content-panel">
        <p>
          Loading artist...
        </p>
      </section>
    )
  }

  if (notFound) {
    return (
      <section className="content-panel">
        <h3>
          Artist not found
        </h3>

        <p>
          This artist does not exist.{' '}
          <Link to="/">
            Back to music
          </Link>
          .
        </p>
      </section>
    )
  }

  if (error) {
    return (
      <section className="content-panel">
        <p>{error}</p>
      </section>
    )
  }

  if (!artist) {
    return null
  }

  const totalPublishedMusic =
    releases.length +
    standaloneTracks.length

  const showHeroVideo =
    Boolean(
      artist.hero_video_url,
    ) &&
    !heroVideoFailed

  const heroBackgroundStyle =
    artist.hero_video_poster_url
      ? {
          backgroundImage:
            `url("${artist.hero_video_poster_url}")`,
        }
      : undefined

  return (
    <>
      <section
        className="artist-hero"
        style={heroBackgroundStyle}
      >
        <div className="artist-hero-media">
          {showHeroVideo && (
            <video
              className="artist-hero-video"
              src={
                artist.hero_video_url
              }
              poster={
                artist.hero_video_poster_url
              }
              autoPlay
              muted
              loop
              playsInline
              preload="metadata"
              aria-hidden="true"
              onError={() =>
                setHeroVideoFailed(
                  true,
                )
              }
            />
          )}

          {!showHeroVideo &&
            artist.hero_video_poster_url && (
              <img
                className="artist-hero-poster"
                src={
                  artist.hero_video_poster_url
                }
                alt=""
                aria-hidden="true"
              />
            )}
        </div>

        <div className="artist-hero-shade" />

        <div className="artist-hero-content">
          <div className="artist-hero-profile">
            <div className="artist-hero-avatar">
              {artist.profile_image_url ? (
                <img
                  src={
                    artist.profile_image_url
                  }
                  alt={`${artist.name} profile`}
                />
              ) : (
                <span>
                  {artist.name
                    .charAt(0)
                    .toUpperCase()}
                </span>
              )}
            </div>

            <div className="artist-hero-details">
              <p className="artist-hero-eyebrow">
                ARTIST
              </p>

              <h1>
                {artist.name}
              </h1>

              {artist.bio && (
                <p className="artist-hero-bio">
                  {artist.bio}
                </p>
              )}

              <div className="artist-hero-meta">
                <span>
                  {
                    totalPublishedMusic
                  }{' '}
                  {totalPublishedMusic ===
                  1
                    ? 'release'
                    : 'releases'}
                </span>

                <span
                  className="artist-hero-meta-dot"
                  aria-hidden="true"
                >
                  •
                </span>

                <span>
                  {followerCount}{' '}
                  {followerCount === 1
                    ? 'follower'
                    : 'followers'}
                </span>

                {tracks.length > 0 && (
                  <>
                    <span
                      className="artist-hero-meta-dot"
                      aria-hidden="true"
                    >
                      •
                    </span>

                    <span>
                      {tracks.length}{' '}
                      {tracks.length === 1
                        ? 'track'
                        : 'tracks'}
                    </span>
                  </>
                )}
              </div>

              <div className="artist-hero-actions">
                <button
                  type="button"
                  className="artist-hero-play-button"
                  disabled={
                    playableTracks.length ===
                    0
                  }
                  onClick={
                    handleHeroPlay
                  }
                >
                  <span
                    aria-hidden="true"
                  >
                    ▶
                  </span>

                  Play
                </button>

                {!isLoadingIdentity &&
                  authenticatedArtist?.id !==
                    artistID && (
                    <button
                      type="button"
                      className="artist-hero-secondary-button"
                      disabled={
                        isFollowLoading
                      }
                      aria-pressed={
                        isFollowing
                      }
                      onClick={() =>
                        void handleFollowToggle()
                      }
                    >
                      {isFollowLoading
                        ? '...'
                        : isFollowing
                          ? 'Following'
                          : 'Follow'}
                    </button>
                  )}

                <button
                  type="button"
                  className="artist-hero-secondary-button"
                  onClick={() =>
                    void handleShare()
                  }
                >
                  Share
                </button>
              </div>

              {shareMessage && (
                <p
                  className="artist-share-message"
                  role="status"
                >
                  {shareMessage}
                </p>
              )}


              {followError && (
                <p
                  className="artist-share-message"
                  role="alert"
                >
                  {followError}
                </p>
              )}
            </div>
          </div>
        </div>
      </section>

      <section className="section">
        <div className="section-heading">
          <div>
            <p className="eyebrow">
              DISCOGRAPHY
            </p>

            <h3>
              Albums & Releases
            </h3>
          </div>
        </div>

        {releases.length === 0 ? (
          <section className="content-panel">
            <p>
              This artist has no
              published albums or
              releases yet.
            </p>
          </section>
        ) : (
          <div className="release-grid">
            {releases.map(
              (release) => {
                const releaseDate =
                  formatReleaseDate(
                    release.release_date,
                  )

                return (
                  <Link
                    to={`/releases/${release.id}`}
                    className="release-card"
                    key={release.id}
                  >
                    <div className="release-card-cover">
                      {release.cover_image_url ? (
                        <img
                          src={
                            release.cover_image_url
                          }
                          alt={`${release.title} cover`}
                        />
                      ) : (
                        <div className="release-cover-fallback">
                          ♪
                        </div>
                      )}
                    </div>

                    <div className="release-card-body">
                      <h4>
                        {
                          release.title
                        }
                      </h4>

                      <p>
                        {
                          release.release_type
                        }

                        {releaseDate
                          ? ` • ${releaseDate}`
                          : ''}
                      </p>
                    </div>
                  </Link>
                )
              },
            )}
          </div>
        )}
      </section>

      <section className="section">
        <div className="section-heading">
          <div>
            <p className="eyebrow">
              MUSIC
            </p>

            <h3>
              Standalone Tracks
            </h3>
          </div>
        </div>

        {standaloneTracks.length ===
          0 && (
          <section className="content-panel">
            <p>
              This artist has no
              standalone tracks.
            </p>
          </section>
        )}

        {standaloneTracks.length >
          0 && (
          <div className="library-list">
            {standaloneTracks.map(
              (track) => {
                const isCurrentTrack =
                  currentTrack?.id ===
                  track.id

                const isThisTrackPlaying =
                  isCurrentTrack &&
                  isPlaying

                const liked =
                  isLiked(track.id)

                const isPendingLike =
                  pendingLikeID ===
                  track.id

                return (
                  <article
                    className="library-track"
                    key={track.id}
                  >
                    <div className="library-track-cover">
                      {track.image_url ? (
                        <img
                          src={
                            track.image_url
                          }
                          alt={`${track.song_title} cover`}
                        />
                      ) : (
                        <span>♪</span>
                      )}
                    </div>

                    <div className="library-track-info">
                      <strong>
                        {
                          track.song_title
                        }
                      </strong>

                      <span>
                        {
                          track.artist_name
                        }
                      </span>

                      {isCurrentTrack && (
                        <p className="music-now-playing">
                          {isPlaying
                            ? 'Now playing'
                            : 'Paused'}
                        </p>
                      )}
                    </div>

                    <span className="genre-pill">
                      {track.genre}
                    </span>

                    <button
                      type="button"
                      className="library-play-button"
                      disabled={
                        !track.audio_url
                      }
                      aria-label={
                        isThisTrackPlaying
                          ? `Pause ${track.song_title}`
                          : `Play ${track.song_title}`
                      }
                      onClick={() =>
                        handlePlay(
                          track,
                        )
                      }
                    >
                      {isThisTrackPlaying
                        ? '❚❚'
                        : '▶'}
                    </button>

                    {isAuthenticated && (
                      <AddToQueueButton
                        track={track}
                        variant="full"
                      />
                    )}

                    {isAuthenticated && (
                      <AddToPlaylistButton
                        track={track}
                      />
                    )}

                    <button
                      type="button"
                      className={
                        liked
                          ? 'like-button is-liked'
                          : 'like-button'
                      }
                      aria-label={
                        liked
                          ? `Unlike ${track.song_title}`
                          : `Like ${track.song_title}`
                      }
                      disabled={
                        isPendingLike
                      }
                      onClick={() =>
                        void handleLike(
                          track,
                        )
                      }
                    >
                      {isPendingLike
                        ? '...'
                        : liked
                          ? '♥'
                          : '♡'}
                    </button>
                  </article>
                )
              },
            )}
          </div>
        )}
      </section>
    </>
  )
}

export default ArtistPage