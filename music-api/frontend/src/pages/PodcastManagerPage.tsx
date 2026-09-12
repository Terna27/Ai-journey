import {
  type FormEvent,
  useCallback,
  useEffect,
  useState,
} from 'react'

import {
  Link,
  useNavigate,
  useParams,
} from 'react-router-dom'

import { useAuth } from '../context/AuthContext'

import PodcastLiveControlPanel from '../components/podcast/PodcastLiveControlPanel'

import {
  createPodcastEpisode,
  deletePodcast,
  deletePodcastEpisode,
  getMyPodcast,
  updatePodcast,
  updatePodcastArtwork,
  updatePodcastEpisode,
  updatePodcastEpisodeMedia,
} from '../lib/api'

import type {
  PodcastDetails,
  PodcastEpisode,
  PodcastEpisodeStatus,
  PodcastEpisodeType,
  PodcastStatus,
} from '../types/podcast'

const MAX_IMAGE_SIZE =
  10 * 1024 * 1024

const MAX_AUDIO_SIZE =
  200 * 1024 * 1024

const ALLOWED_IMAGE_TYPES = [
  'image/jpeg',
  'image/png',
  'image/webp',
]

const ALLOWED_AUDIO_TYPES = [
  'audio/mpeg',
  'audio/wav',
  'audio/x-wav',
  'audio/ogg',
  'audio/mp4',
  'audio/x-m4a',
  'audio/aac',
  'video/mp4',
  'video/webm',
]

function validateImage(file: File) {
  if (
    !ALLOWED_IMAGE_TYPES.includes(
      file.type,
    )
  ) {
    return 'Artwork must be JPEG, PNG, or WebP.'
  }

  if (file.size > MAX_IMAGE_SIZE) {
    return 'Artwork must not exceed 10 MB.'
  }

  return ''
}

function validateAudio(file: File) {
  if (
    file.size > MAX_AUDIO_SIZE
  ) {
    return 'Episode audio must not exceed 200 MB.'
  }

  if (
    file.type &&
    !ALLOWED_AUDIO_TYPES.includes(
      file.type,
    )
  ) {
    return 'Audio must be MP3, WAV, OGG, M4A, AAC, MP4, or WebM.'
  }

  return ''
}

function formatDuration(
  durationMS: number,
) {
  if (
    !Number.isFinite(durationMS) ||
    durationMS <= 0
  ) {
    return 'Duration unavailable'
  }

  const totalSeconds =
    Math.floor(durationMS / 1000)

  const hours =
    Math.floor(
      totalSeconds / 3600,
    )

  const minutes =
    Math.floor(
      (totalSeconds % 3600) /
        60,
    )

  const seconds =
    totalSeconds % 60

  if (hours > 0) {
    return [
      hours,
      String(minutes).padStart(
        2,
        '0',
      ),
      String(seconds).padStart(
        2,
        '0',
      ),
    ].join(':')
  }

  return [
    minutes,
    String(seconds).padStart(
      2,
      '0',
    ),
  ].join(':')
}

function PodcastManagerPage() {
  const navigate = useNavigate()

  const { id } = useParams()

  const { token } = useAuth()

  const podcastID = Number(id)

  const validPodcastID =
    Number.isInteger(podcastID) &&
    podcastID > 0

  const [details, setDetails] =
    useState<PodcastDetails | null>(
      null,
    )

  const [loading, setLoading] =
    useState(true)

  const [notFound, setNotFound] =
    useState(false)

  const [error, setError] =
    useState('')

  const [success, setSuccess] =
    useState('')

  const [saving, setSaving] =
    useState(false)

  const [
    uploadingArtwork,
    setUploadingArtwork,
  ] = useState(false)

  const [deleting, setDeleting] =
    useState(false)

  const [
    creatingEpisode,
    setCreatingEpisode,
  ] = useState(false)

  const [
    workingEpisodeID,
    setWorkingEpisodeID,
  ] = useState<number | null>(
    null,
  )

  const [title, setTitle] =
    useState('')

  const [description, setDescription] =
    useState('')

  const [category, setCategory] =
    useState('')

  const [isExplicit, setIsExplicit] =
    useState(false)

  const [
    episodeTitle,
    setEpisodeTitle,
  ] = useState('')

  const [
    episodeDescription,
    setEpisodeDescription,
  ] = useState('')

  const [
    episodeSeason,
    setEpisodeSeason,
  ] = useState('1')

  const [
    episodeNumber,
    setEpisodeNumber,
  ] = useState('')

  const [
    episodeType,
    setEpisodeType,
  ] =
    useState<PodcastEpisodeType>(
      'FULL',
    )

  const [
    episodeExplicit,
    setEpisodeExplicit,
  ] = useState(false)

  const applyDetails = useCallback(
    (next: PodcastDetails) => {
      setDetails(next)

      setTitle(next.podcast.title)

      setDescription(
        next.podcast.description,
      )

      setCategory(
        next.podcast.category,
      )

      setIsExplicit(
        next.podcast.is_explicit,
      )
    },
    [],
  )

  const refreshPodcast =
    useCallback(async () => {
      if (
        !token ||
        !validPodcastID
      ) {
        return
      }

      const result =
        await getMyPodcast(
          podcastID,
          token,
        )

      applyDetails(result)
    }, [
      applyDetails,
      podcastID,
      token,
      validPodcastID,
    ])

  useEffect(() => {
    if (
      !token ||
      !validPodcastID
    ) {
      setLoading(false)
      return
    }

    const accessToken = token

    let cancelled = false

    async function load() {
      setLoading(true)
      setError('')
      setNotFound(false)

      try {
        const result =
          await getMyPodcast(
            podcastID,
            accessToken,
          )

        if (!cancelled) {
          applyDetails(result)
        }
      } catch (err) {
        if (cancelled) {
          return
        }

        const message =
          err instanceof Error
            ? err.message
            : 'Unable to load podcast.'

        if (
          message
            .toLowerCase()
            .includes('not found')
        ) {
          setNotFound(true)
        } else {
          setError(message)
        }
      } finally {
        if (!cancelled) {
          setLoading(false)
        }
      }
    }

    void load()

    return () => {
      cancelled = true
    }
  }, [
    applyDetails,
    podcastID,
    token,
    validPodcastID,
  ])

  async function savePodcast(
    status?: PodcastStatus,
  ) {
    if (
      !token ||
      !details ||
      saving
    ) {
      return
    }

    if (!title.trim()) {
      setError(
        'Podcast title is required.',
      )
      return
    }

    if (
      status === 'PUBLISHED' &&
      !details.podcast.artwork_url
    ) {
      setError(
        'Upload podcast artwork before publishing.',
      )
      return
    }

    setSaving(true)
    setError('')
    setSuccess('')

    try {
      await updatePodcast(
        podcastID,
        {
          title: title.trim(),
          description:
            description.trim(),
          category: category.trim(),
          is_explicit: isExplicit,
          ...(status
            ? { status }
            : {}),
        },
        token,
      )

      await refreshPodcast()

      setSuccess(
        status === 'PUBLISHED'
          ? 'Podcast published successfully.'
          : status === 'ARCHIVED'
            ? 'Podcast archived.'
            : status === 'DRAFT'
              ? 'Podcast returned to draft.'
              : 'Podcast details saved.',
      )
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : 'Unable to save podcast.',
      )
    } finally {
      setSaving(false)
    }
  }

  async function handleSavePodcast(
    event: FormEvent<HTMLFormElement>,
  ) {
    event.preventDefault()

    await savePodcast()
  }

  async function handleArtwork(
    file: File | null,
  ) {
    if (
      !file ||
      !token ||
      uploadingArtwork
    ) {
      return
    }

    const validationError =
      validateImage(file)

    if (validationError) {
      setError(validationError)
      return
    }

    setUploadingArtwork(true)
    setError('')
    setSuccess('')

    try {
      await updatePodcastArtwork(
        podcastID,
        { artwork: file },
        token,
      )

      await refreshPodcast()

      setSuccess(
        'Podcast artwork updated.',
      )
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : 'Unable to upload artwork.',
      )
    } finally {
      setUploadingArtwork(false)
    }
  }

  async function handleCreateEpisode(
    event: FormEvent<HTMLFormElement>,
  ) {
    event.preventDefault()

    if (
      !token ||
      creatingEpisode
    ) {
      return
    }

    const accessToken = token

    const cleanTitle =
      episodeTitle.trim()

    if (!cleanTitle) {
      setError(
        'Episode title is required.',
      )
      return
    }

    const season =
      Number(episodeSeason)

    if (
      !Number.isInteger(season) ||
      season <= 0
    ) {
      setError(
        'Season number must be a positive whole number.',
      )
      return
    }

    let number:
      number | undefined

    if (episodeNumber.trim()) {
      number =
        Number(episodeNumber)

      if (
        !Number.isInteger(number) ||
        number <= 0
      ) {
        setError(
          'Episode number must be a positive whole number.',
        )
        return
      }
    }

    setCreatingEpisode(true)
    setError('')
    setSuccess('')

    try {
      await createPodcastEpisode(
        podcastID,
        {
          title: cleanTitle,
          description:
            episodeDescription.trim(),
          season_number: season,
          ...(number !== undefined
            ? {
                episode_number:
                  number,
              }
            : {}),
          episode_type:
            episodeType,
          is_explicit:
            episodeExplicit,
          status: 'DRAFT',
        },
        accessToken,
      )

      setEpisodeTitle('')
      setEpisodeDescription('')
      setEpisodeSeason('1')
      setEpisodeNumber('')
      setEpisodeType('FULL')
      setEpisodeExplicit(false)

      await refreshPodcast()

      setSuccess(
        'Episode created as a draft.',
      )
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : 'Unable to create episode.',
      )
    } finally {
      setCreatingEpisode(false)
    }
  }

  async function changeEpisodeStatus(
    episode: PodcastEpisode,
    status: PodcastEpisodeStatus,
  ) {
    if (
      !token ||
      workingEpisodeID !== null
    ) {
      return
    }

    if (
      status === 'PUBLISHED' &&
      !episode.audio_url
    ) {
      setError(
        'Upload episode audio before publishing.',
      )
      return
    }

    setWorkingEpisodeID(
      episode.id,
    )

    setError('')
    setSuccess('')

    try {
      await updatePodcastEpisode(
        episode.id,
        { status },
        token,
      )

      await refreshPodcast()

      setSuccess(
        status === 'PUBLISHED'
          ? `"${episode.title}" published.`
          : status === 'ARCHIVED'
            ? `"${episode.title}" archived.`
            : `"${episode.title}" returned to draft.`,
      )
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : 'Unable to update episode.',
      )
    } finally {
      setWorkingEpisodeID(
        null,
      )
    }
  }

  async function uploadEpisodeMedia(
    episode: PodcastEpisode,
    audio?: File,
    artwork?: File,
  ) {
    if (
      !token ||
      workingEpisodeID !== null
    ) {
      return
    }

    if (!audio && !artwork) {
      return
    }

    if (audio) {
      const validationError =
        validateAudio(audio)

      if (validationError) {
        setError(validationError)
        return
      }
    }

    if (artwork) {
      const validationError =
        validateImage(artwork)

      if (validationError) {
        setError(validationError)
        return
      }
    }

    setWorkingEpisodeID(
      episode.id,
    )

    setError('')
    setSuccess('')

    try {
      await updatePodcastEpisodeMedia(
        episode.id,
        {
          ...(audio
            ? { audio }
            : {}),
          ...(artwork
            ? { artwork }
            : {}),
        },
        token,
      )

      await refreshPodcast()

      setSuccess(
        `"${episode.title}" media updated.`,
      )
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : 'Unable to upload episode media.',
      )
    } finally {
      setWorkingEpisodeID(
        null,
      )
    }
  }

  async function handleDeleteEpisode(
    episode: PodcastEpisode,
  ) {
    if (
      !token ||
      workingEpisodeID !== null
    ) {
      return
    }

    const confirmed =
      window.confirm(
        `Delete "${episode.title}"? Its uploaded media will also be removed.`,
      )

    if (!confirmed) {
      return
    }

    setWorkingEpisodeID(
      episode.id,
    )

    setError('')
    setSuccess('')

    try {
      await deletePodcastEpisode(
        episode.id,
        token,
      )

      await refreshPodcast()

      setSuccess(
        `"${episode.title}" deleted.`,
      )
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : 'Unable to delete episode.',
      )
    } finally {
      setWorkingEpisodeID(
        null,
      )
    }
  }

  async function handleDeletePodcast() {
    if (
      !token ||
      !details ||
      deleting
    ) {
      return
    }

    const confirmed =
      window.confirm(
        `Delete "${details.podcast.title}" and all of its episodes? This cannot be undone.`,
      )

    if (!confirmed) {
      return
    }

    setDeleting(true)
    setError('')

    try {
      await deletePodcast(
        podcastID,
        token,
      )

      navigate(
        '/my-podcasts',
        { replace: true },
      )
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : 'Unable to delete podcast.',
      )
    } finally {
      setDeleting(false)
    }
  }

  if (!validPodcastID) {
    return (
      <section className="content-panel">
        <h3>Podcast not found</h3>

        <Link to="/my-podcasts">
          Back to My Podcasts
        </Link>
      </section>
    )
  }

  if (loading) {
    return (
      <section className="content-panel">
        <p>Loading podcast...</p>
      </section>
    )
  }

  if (notFound) {
    return (
      <section className="content-panel">
        <h3>Podcast not found</h3>

        <p>
          This podcast does not exist
          or does not belong to your
          account.
        </p>

        <Link to="/my-podcasts">
          Back to My Podcasts
        </Link>
      </section>
    )
  }

  if (!details) {
    return (
      <section className="content-panel">
        <p>
          {error ||
            'Unable to load podcast.'}
        </p>
      </section>
    )
  }

  const { podcast, episodes } =
    details

  return (
    <>
      <header className="page-header podcast-owner-header">
        <div>
          <p className="eyebrow">
            PODCAST STUDIO
          </p>

          <h2>{podcast.title}</h2>

          <p>
            {podcast.status}
            {' • '}
            {episodes.length}{' '}
            {episodes.length === 1
              ? 'episode'
              : 'episodes'}
          </p>
        </div>

        <div className="release-manager-header-actions">
          {podcast.status ===
            'PUBLISHED' && (
            <Link
              to={`/podcasts/${encodeURIComponent(
                podcast.slug,
              )}`}
              className="release-manager-link"
            >
              View Public Page
            </Link>
          )}

          <Link
            to="/my-podcasts"
            className="release-manager-link"
          >
            All Podcasts
          </Link>
        </div>
      </header>

      {error && (
        <div
          className="status-message"
          role="alert"
        >
          {error}
        </div>
      )}

      {success && (
        <div
          className="status-message success-message"
          role="status"
        >
          {success}
        </div>
      )}

      <section className="content-panel">
        <div className="section-heading">
          <div>
            <p className="eyebrow">
              SHOW DETAILS
            </p>

            <h3>
              Podcast information
            </h3>
          </div>
        </div>

        <div className="podcast-manager-artwork-row">
          <div className="podcast-manager-artwork">
            {podcast.artwork_url ? (
              <img
                src={
                  podcast.artwork_url
                }
                alt={`${podcast.title} artwork`}
              />
            ) : (
              <span>◉</span>
            )}
          </div>

          <label className="podcast-file-button">
            <span>
              {uploadingArtwork
                ? 'Uploading...'
                : podcast.artwork_url
                  ? 'Change Artwork'
                  : 'Upload Artwork'}
            </span>

            <input
              type="file"
              accept="image/jpeg,image/png,image/webp"
              disabled={
                uploadingArtwork
              }
              onChange={(event) => {
                const file =
                  event.target
                    .files?.[0] ??
                  null

                void handleArtwork(
                  file,
                )

                event.target.value =
                  ''
              }}
            />
          </label>
        </div>

        <form
          className="release-manager-form"
          onSubmit={
            handleSavePodcast
          }
        >
          <label>
            <span>Title</span>

            <input
              type="text"
              value={title}
              onChange={(event) =>
                setTitle(
                  event.target.value,
                )
              }
              required
            />
          </label>

          <label>
            <span>Category</span>

            <input
              type="text"
              value={category}
              onChange={(event) =>
                setCategory(
                  event.target.value,
                )
              }
            />
          </label>

          <label className="release-form-wide">
            <span>Description</span>

            <textarea
              value={description}
              onChange={(event) =>
                setDescription(
                  event.target.value,
                )
              }
              rows={5}
            />
          </label>

          <label className="podcast-checkbox-field release-form-wide">
            <input
              type="checkbox"
              checked={isExplicit}
              onChange={(event) =>
                setIsExplicit(
                  event.target.checked,
                )
              }
            />

            <span>
              Explicit content
            </span>
          </label>

          <div className="release-form-wide release-edit-actions">
            <button
              type="submit"
              className="release-primary-button"
              disabled={saving}
            >
              {saving
                ? 'Saving...'
                : 'Save Changes'}
            </button>

            {podcast.status !==
              'PUBLISHED' && (
              <button
                type="button"
                className="release-publish-button"
                disabled={saving}
                onClick={() =>
                  void savePodcast(
                    'PUBLISHED',
                  )
                }
              >
                Publish Podcast
              </button>
            )}

            {podcast.status ===
              'PUBLISHED' && (
              <button
                type="button"
                className="release-secondary-button"
                disabled={saving}
                onClick={() =>
                  void savePodcast(
                    'DRAFT',
                  )
                }
              >
                Return to Draft
              </button>
            )}

            {podcast.status !==
              'ARCHIVED' && (
              <button
                type="button"
                className="release-secondary-button"
                disabled={saving}
                onClick={() =>
                  void savePodcast(
                    'ARCHIVED',
                  )
                }
              >
                Archive
              </button>
            )}
          </div>
        </form>
      </section>

      <section className="section">
        <div className="section-heading">
          <div>
            <p className="eyebrow">
              NEW EPISODE
            </p>

            <h3>Create episode</h3>
          </div>
        </div>

        <section className="content-panel">
          <form
            className="release-manager-form"
            onSubmit={
              handleCreateEpisode
            }
          >
            <label>
              <span>Title</span>

              <input
                type="text"
                value={episodeTitle}
                onChange={(event) =>
                  setEpisodeTitle(
                    event.target.value,
                  )
                }
                required
              />
            </label>

            <label>
              <span>Episode type</span>

              <select
                value={episodeType}
                onChange={(event) =>
                  setEpisodeType(
                    event.target
                      .value as PodcastEpisodeType,
                  )
                }
              >
                <option value="FULL">
                  Full Episode
                </option>

                <option value="TRAILER">
                  Trailer
                </option>

                <option value="BONUS">
                  Bonus
                </option>
              </select>
            </label>

            <label>
              <span>Season</span>

              <input
                type="number"
                min="1"
                step="1"
                value={episodeSeason}
                onChange={(event) =>
                  setEpisodeSeason(
                    event.target.value,
                  )
                }
                required
              />
            </label>

            <label>
              <span>
                Episode number
              </span>

              <input
                type="number"
                min="1"
                step="1"
                value={episodeNumber}
                onChange={(event) =>
                  setEpisodeNumber(
                    event.target.value,
                  )
                }
                placeholder="Optional"
              />
            </label>

            <label className="release-form-wide">
              <span>Description</span>

              <textarea
                value={
                  episodeDescription
                }
                onChange={(event) =>
                  setEpisodeDescription(
                    event.target.value,
                  )
                }
                rows={4}
              />
            </label>

            <label className="podcast-checkbox-field release-form-wide">
              <input
                type="checkbox"
                checked={
                  episodeExplicit
                }
                onChange={(event) =>
                  setEpisodeExplicit(
                    event.target.checked,
                  )
                }
              />

              <span>
                Explicit content
              </span>
            </label>

            <div className="release-form-wide release-edit-actions">
              <button
                type="submit"
                className="release-primary-button"
                disabled={
                  creatingEpisode
                }
              >
                {creatingEpisode
                  ? 'Creating...'
                  : 'Create Draft Episode'}
              </button>
            </div>
          </form>
        </section>
      </section>

      <section className="section">
        <div className="section-heading">
          <div>
            <p className="eyebrow">
              EPISODES
            </p>

            <h3>
              Manage episodes
            </h3>
          </div>
        </div>

        {episodes.length === 0 ? (
          <section className="content-panel">
            <p>
              No episodes yet. Create
              your first episode above.
            </p>
          </section>
        ) : (
          <div className="podcast-episode-manager-list">
            {episodes.map(
              (episode) => {
                const working =
                  workingEpisodeID ===
                  episode.id

                return (
                  <article
                    className="podcast-episode-manager-card"
                    key={episode.id}
                  >
                    <div className="podcast-episode-manager-artwork">
                      {episode.artwork_url ||
                      podcast.artwork_url ? (
                        <img
                          src={
                            episode.artwork_url ||
                            podcast.artwork_url
                          }
                          alt={`${episode.title} artwork`}
                        />
                      ) : (
                        <span>♪</span>
                      )}
                    </div>

                    <div className="podcast-episode-manager-body">
                      <div className="podcast-episode-manager-heading">
                        <div>
                          <span className={`podcast-status podcast-status-${episode.status.toLowerCase()}`}>
                            {episode.status}
                          </span>

                          <h3>
                            {
                              episode.title
                            }
                          </h3>
                        </div>

                        {episode.is_explicit && (
                          <span className="podcast-explicit-badge">
                            E
                          </span>
                        )}
                      </div>

                      <p className="podcast-owner-meta">
                        Season{' '}
                        {
                          episode.season_number
                        }

                        {episode.episode_number !==
                          null &&
                          ` • Episode ${episode.episode_number}`}

                        {' • '}
                        {
                          episode.episode_type
                        }

                        {' • '}
                        {formatDuration(
                          episode.duration_ms,
                        )}
                      </p>

                      <p className="podcast-owner-description">
                        {episode.description ||
                          'No description yet.'}
                      </p>

                      <div className="podcast-media-status">
                        <span>
                          Audio:{' '}
                          {episode.audio_url
                            ? 'Ready'
                            : 'Missing'}
                        </span>

                        <span>
                          Artwork:{' '}
                          {episode.artwork_url
                            ? 'Ready'
                            : 'Using show artwork'}
                        </span>
                      </div>

                      <div className="podcast-episode-file-actions">
                        <label className="podcast-file-button">
                          <span>
                            {working
                              ? 'Working...'
                              : episode.audio_url
                                ? 'Replace Audio'
                                : 'Upload Audio'}
                          </span>

                          <input
                            type="file"
                            accept="audio/*,.m4a,.aac,.mp4,.webm"
                            disabled={
                              working
                            }
                            onChange={(
                              event,
                            ) => {
                              const file =
                                event
                                  .target
                                  .files?.[0] ??
                                undefined

                              if (file) {
                                void uploadEpisodeMedia(
                                  episode,
                                  file,
                                )
                              }

                              event.target.value =
                                ''
                            }}
                          />
                        </label>

                        <label className="podcast-file-button">
                          <span>
                            {working
                              ? 'Working...'
                              : episode.artwork_url
                                ? 'Replace Artwork'
                                : 'Episode Artwork'}
                          </span>

                          <input
                            type="file"
                            accept="image/jpeg,image/png,image/webp"
                            disabled={
                              working
                            }
                            onChange={(
                              event,
                            ) => {
                              const file =
                                event
                                  .target
                                  .files?.[0] ??
                                undefined

                              if (file) {
                                void uploadEpisodeMedia(
                                  episode,
                                  undefined,
                                  file,
                                )
                              }

                              event.target.value =
                                ''
                            }}
                          />
                        </label>
                      </div>

                      <div className="podcast-owner-actions">
                        {/* Live-managed statuses (SCHEDULED,
                            LIVE, ENDED) change only through
                            the live control panel below. */}
                        {(episode.status ===
                          'DRAFT' ||
                          episode.status ===
                            'ARCHIVED') && (
                          <button
                            type="button"
                            className="release-publish-button"
                            disabled={
                              working
                            }
                            onClick={() =>
                              void changeEpisodeStatus(
                                episode,
                                'PUBLISHED',
                              )
                            }
                          >
                            Publish
                          </button>
                        )}

                        {episode.status ===
                          'PUBLISHED' && (
                          <button
                            type="button"
                            className="release-secondary-button"
                            disabled={
                              working
                            }
                            onClick={() =>
                              void changeEpisodeStatus(
                                episode,
                                'DRAFT',
                              )
                            }
                          >
                            Return to Draft
                          </button>
                        )}

                        {(episode.status ===
                          'DRAFT' ||
                          episode.status ===
                            'PUBLISHED') && (
                          <button
                            type="button"
                            className="release-secondary-button"
                            disabled={
                              working
                            }
                            onClick={() =>
                              void changeEpisodeStatus(
                                episode,
                                'ARCHIVED',
                              )
                            }
                          >
                            Archive
                          </button>
                        )}

                        <button
                          type="button"
                          className="release-delete-button"
                          disabled={
                            working
                          }
                          onClick={() =>
                            void handleDeleteEpisode(
                              episode,
                            )
                          }
                        >
                          Delete Episode
                        </button>
                      </div>

                      {token ? (
                        <PodcastLiveControlPanel
                          episode={
                            episode
                          }
                          token={
                            token
                          }
                          onRefresh={
                            refreshPodcast
                          }
                        />
                      ) : null}
                    </div>
                  </article>
                )
              },
            )}
          </div>
        )}
      </section>

      <section className="content-panel release-danger-zone">
        <div>
          <p className="eyebrow">
            DANGER ZONE
          </p>

          <h3>Delete podcast</h3>

          <p>
            This permanently deletes
            the podcast, all episodes,
            and their uploaded media.
          </p>
        </div>

        <button
          type="button"
          className="release-delete-button"
          disabled={deleting}
          onClick={() =>
            void handleDeletePodcast()
          }
        >
          {deleting
            ? 'Deleting...'
            : 'Delete Podcast'}
        </button>
      </section>
    </>
  )
}

export default PodcastManagerPage
