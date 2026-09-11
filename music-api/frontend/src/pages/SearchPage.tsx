import {
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react'

import type {
  SyntheticEvent,
} from 'react'

import {
  Link,
  useNavigate,
  useSearchParams,
} from 'react-router-dom'

import AddToPlaylistButton from '../components/music/AddToPlaylistButton'
import AddToQueueButton from '../components/music/AddToQueueButton'

import { useAuth } from '../context/AuthContext'
import { useLibrary } from '../context/LibraryContext'
import { usePlayer } from '../context/PlayerContext'

import { searchMusic } from '../lib/api'

import type { Music } from '../types/music'

import type {
  SearchResponse,
  SearchSort,
  SearchType,
} from '../types/search'

const SEARCH_LIMIT = 10
const SUGGESTION_LIMIT = 5
const SUGGESTION_DELAY_MS = 300

function isSearchType(
  value: string | null,
): value is SearchType {
  return (
    value === 'all' ||
    value === 'track' ||
    value === 'artist' ||
    value === 'release'
  )
}

function isSearchSort(
  value: string | null,
): value is SearchSort {
  return (
    value === 'relevance' ||
    value === 'newest' ||
    value === 'popular'
  )
}

function SearchPage() {
  const navigate = useNavigate()

  const [
    searchParams,
    setSearchParams,
  ] = useSearchParams()

  const { isAuthenticated } =
    useAuth()

  const {
    isLiked,
    toggleLike,
  } = useLibrary()

  const {
    currentTrack,
    isPlaying,
    playTrack,
  } = usePlayer()

  const searchBoxRef =
    useRef<HTMLDivElement | null>(
      null,
    )

  const suggestionRequestRef =
    useRef(0)

  const query =
    searchParams
      .get('q')
      ?.trim() ?? ''

  const rawType =
    searchParams.get('type')

  const type: SearchType =
    isSearchType(rawType)
      ? rawType
      : 'all'

  const rawSort =
    searchParams.get('sort')

  const sort: SearchSort =
    isSearchSort(rawSort)
      ? rawSort
      : 'relevance'

  const genre =
    searchParams
      .get('genre')
      ?.trim() ?? ''

  const rawPage =
    Number(
      searchParams.get('page') ??
        '1',
    )

  const page =
    Number.isInteger(rawPage) &&
    rawPage > 0
      ? rawPage
      : 1

  const [
    searchInput,
    setSearchInput,
  ] = useState(query)

  const [
    result,
    setResult,
  ] = useState<SearchResponse | null>(
    null,
  )

  const [
    suggestions,
    setSuggestions,
  ] = useState<Music[]>([])

  const [
    suggestionsLoading,
    setSuggestionsLoading,
  ] = useState(false)

  const [
    suggestionsOpen,
    setSuggestionsOpen,
  ] = useState(false)

  const [
    activeSuggestionIndex,
    setActiveSuggestionIndex,
  ] = useState(-1)

  const [loading, setLoading] =
    useState(false)

  const [error, setError] =
    useState('')

  const [
    pendingLikeTrackID,
    setPendingLikeTrackID,
  ] = useState<number | null>(null)

  const [
    likeError,
    setLikeError,
  ] = useState('')

  useEffect(() => {
    setSearchInput(query)
  }, [query])

  useEffect(() => {
    function handleDocumentPointerDown(
      event: PointerEvent,
    ) {
      const target =
        event.target

      if (
        !(target instanceof Node)
      ) {
        return
      }

      if (
        searchBoxRef.current &&
        !searchBoxRef.current.contains(
          target,
        )
      ) {
        setSuggestionsOpen(false)

        setActiveSuggestionIndex(
          -1,
        )
      }
    }

    document.addEventListener(
      'pointerdown',
      handleDocumentPointerDown,
    )

    return () => {
      document.removeEventListener(
        'pointerdown',
        handleDocumentPointerDown,
      )
    }
  }, [])

  useEffect(() => {
    const term =
      searchInput.trim()

    const requestID =
      ++suggestionRequestRef.current

    setActiveSuggestionIndex(-1)

    if (term.length < 2) {
      setSuggestions([])
      setSuggestionsLoading(false)
      setSuggestionsOpen(false)

      return
    }

    const timer =
      window.setTimeout(
        () => {
          async function loadSuggestions() {
            try {
              setSuggestionsLoading(true)

              const response =
                await searchMusic({
                  query: term,
                  type: 'track',
                  sort: 'relevance',
                  page: 1,
                  limit: SUGGESTION_LIMIT,
                })

              if (
                requestID !==
                suggestionRequestRef.current
              ) {
                return
              }

              const normalizedTerm =
                term.toLocaleLowerCase()

              const titleMatches =
                response.tracks
                  .filter((track) =>
                    track.song_title
                      .toLocaleLowerCase()
                      .includes(
                        normalizedTerm,
                      ),
                  )
                  .slice(
                    0,
                    SUGGESTION_LIMIT,
                  )

              setSuggestions(
                titleMatches,
              )

              setSuggestionsOpen(
                true,
              )
            } catch {
              if (
                requestID !==
                suggestionRequestRef.current
              ) {
                return
              }

              setSuggestions([])
              setSuggestionsOpen(false)
            } finally {
              if (
                requestID ===
                suggestionRequestRef.current
              ) {
                setSuggestionsLoading(
                  false,
                )
              }
            }
          }

          void loadSuggestions()
        },
        SUGGESTION_DELAY_MS,
      )

    return () => {
      window.clearTimeout(timer)
    }
  }, [searchInput])

  useEffect(() => {
    let cancelled = false

    if (!query) {
      setResult(null)
      setError('')
      setLoading(false)

      return () => {
        cancelled = true
      }
    }

    if (query.length < 2) {
      setResult(null)

      setError(
        'Enter at least 2 characters to search.',
      )

      setLoading(false)

      return () => {
        cancelled = true
      }
    }

    async function loadSearch() {
      try {
        setLoading(true)
        setError('')

        const response =
          await searchMusic({
            query,
            type,
            sort,
            genre:
              type === 'track' ||
              type === 'all'
                ? genre
                : '',
            page,
            limit: SEARCH_LIMIT,
          })

        if (!cancelled) {
          setResult(response)
        }
      } catch (err) {
        if (!cancelled) {
          setResult(null)

          setError(
            err instanceof Error
              ? err.message
              : 'Search failed',
          )
        }
      } finally {
        if (!cancelled) {
          setLoading(false)
        }
      }
    }

    void loadSearch()

    return () => {
      cancelled = true
    }
  }, [
    query,
    type,
    sort,
    genre,
    page,
  ])

  const totalResults =
    useMemo(() => {
      if (!result) {
        return 0
      }

      return (
        result.pagination
          .track_total +
        result.pagination
          .artist_total +
        result.pagination
          .release_total
      )
    }, [result])

  function updateSearchParams(
    updates: Record<
      string,
      string | null
    >,
  ) {
    const next =
      new URLSearchParams(
        searchParams,
      )

    for (const [
      key,
      value,
    ] of Object.entries(updates)) {
      if (
        value === null ||
        value === ''
      ) {
        next.delete(key)
      } else {
        next.set(key, value)
      }
    }

    setSearchParams(next)
  }

  function performSearch(
    value: string,
  ) {
    const nextQuery =
      value.trim()

    if (nextQuery.length < 2) {
      setError(
        'Enter at least 2 characters to search.',
      )

      return
    }

    setSuggestionsOpen(false)
    setActiveSuggestionIndex(-1)

    updateSearchParams({
      q: nextQuery,
      page: '1',
    })
  }

  function handleSubmit(
    event: SyntheticEvent<HTMLFormElement>,
  ) {
    event.preventDefault()

    if (
      suggestionsOpen &&
      activeSuggestionIndex >= 0 &&
      suggestions[
        activeSuggestionIndex
      ]
    ) {
      const selected =
        suggestions[
          activeSuggestionIndex
        ]

      setSearchInput(
        selected.song_title,
      )

      performSearch(
        selected.song_title,
      )

      return
    }

    performSearch(searchInput)
  }

  function handleSuggestionSelect(
    track: Music,
  ) {
    setSearchInput(
      track.song_title,
    )

    setSuggestions([])
    setSuggestionsOpen(false)
    setActiveSuggestionIndex(-1)

    updateSearchParams({
      q: track.song_title,
      type: 'track',
      page: '1',
    })
  }

  function handleTypeChange(
    nextType: SearchType,
  ) {
    updateSearchParams({
      type:
        nextType === 'all'
          ? null
          : nextType,
      genre:
        nextType === 'artist' ||
        nextType === 'release'
          ? null
          : genre || null,
      page: '1',
    })
  }

  function handleSortChange(
    nextSort: SearchSort,
  ) {
    updateSearchParams({
      sort:
        nextSort === 'relevance'
          ? null
          : nextSort,
      page: '1',
    })
  }

  function handleGenreChange(
    value: string,
  ) {
    updateSearchParams({
      genre:
        value.trim() || null,
      page: '1',
    })
  }

  function handlePageChange(
    nextPage: number,
  ) {
    if (nextPage < 1) {
      return
    }

    updateSearchParams({
      page:
        nextPage === 1
          ? null
          : String(nextPage),
    })

    window.scrollTo({
      top: 0,
      behavior: 'smooth',
    })
  }

  function handlePlay(
    track: Music,
  ) {
    if (!isAuthenticated) {
      navigate('/login', {
        state: {
          message:
            'Please log in to play music.',
        },
      })

      return
    }

    playTrack(
      track,
      result?.tracks ?? [track],
    )
  }

  async function handleLike(
    track: Music,
  ) {
    if (!isAuthenticated) {
      navigate('/login', {
        state: {
          message:
            'Please log in to like music.',
        },
      })

      return
    }

    if (
      pendingLikeTrackID !== null
    ) {
      return
    }

    try {
      setPendingLikeTrackID(
        track.id,
      )

      setLikeError('')

      await toggleLike(track)
    } catch (err) {
      setLikeError(
        err instanceof Error
          ? err.message
          : 'Failed to update liked songs',
      )
    } finally {
      setPendingLikeTrackID(null)
    }
  }

  const trackHasMore =
    result?.pagination
      .track_has_more ?? false

  const artistHasMore =
    result?.pagination
      .artist_has_more ?? false

  const releaseHasMore =
    result?.pagination
      .release_has_more ?? false

  const hasMore =
    type === 'track'
      ? trackHasMore
      : type === 'artist'
        ? artistHasMore
        : type === 'release'
          ? releaseHasMore
          : (
              trackHasMore ||
              artistHasMore ||
              releaseHasMore
            )

  return (
    <>
      <header className="search-page-header">
        <div>
          <p className="eyebrow">
            SEARCH
          </p>

          <h2>
            Find your next sound
          </h2>

          <p className="search-page-intro">
            Search tracks, artists and
            releases across Music.
          </p>
        </div>
      </header>

      <section className="search-panel">
        <form
          className="search-form"
          onSubmit={handleSubmit}
        >
          <div
            className="search-autocomplete"
            ref={searchBoxRef}
          >
            <input
              type="search"
              value={searchInput}
              onChange={(event) => {
                setSearchInput(
                  event.target.value,
                )

                setSuggestionsOpen(
                  event.target.value
                    .trim()
                    .length >= 2,
                )
              }}
              onFocus={() => {
                if (
                  searchInput
                    .trim()
                    .length >= 2
                ) {
                  setSuggestionsOpen(
                    true,
                  )
                }
              }}
              onKeyDown={(event) => {
                if (
                  event.key ===
                  'Escape'
                ) {
                  setSuggestionsOpen(
                    false,
                  )

                  setActiveSuggestionIndex(
                    -1,
                  )

                  return
                }

                if (
                  !suggestionsOpen ||
                  suggestions.length ===
                    0
                ) {
                  return
                }

                if (
                  event.key ===
                  'ArrowDown'
                ) {
                  event.preventDefault()

                  setActiveSuggestionIndex(
                    (current) => {
                      if (
                        current >=
                        suggestions.length -
                          1
                      ) {
                        return 0
                      }

                      return current + 1
                    },
                  )

                  return
                }

                if (
                  event.key ===
                  'ArrowUp'
                ) {
                  event.preventDefault()

                  setActiveSuggestionIndex(
                    (current) => {
                      if (
                        current <= 0
                      ) {
                        return (
                          suggestions.length -
                          1
                        )
                      }

                      return current - 1
                    },
                  )
                }
              }}
              placeholder="Search songs, artists, genres or releases"
              aria-label="Search music"
              aria-autocomplete="list"
              aria-expanded={
                suggestionsOpen
              }
              aria-controls="search-suggestions"
            />

            {suggestionsOpen && (
              <div
                id="search-suggestions"
                className="search-suggestions"
                role="listbox"
              >
                {suggestionsLoading && (
                  <div className="search-suggestion-status">
                    Searching songs...
                  </div>
                )}

                {!suggestionsLoading &&
                  suggestions.length ===
                    0 && (
                    <div className="search-suggestion-status">
                      No matching song
                      titles found.
                    </div>
                  )}

                {!suggestionsLoading &&
                  suggestions.map(
                    (
                      track,
                      index,
                    ) => (
                      <button
                        key={track.id}
                        type="button"
                        role="option"
                        aria-selected={
                          activeSuggestionIndex ===
                          index
                        }
                        className={`search-suggestion-item${
                          activeSuggestionIndex ===
                          index
                            ? ' active'
                            : ''
                        }`}
                        onMouseEnter={() =>
                          setActiveSuggestionIndex(
                            index,
                          )
                        }
                        onClick={() =>
                          handleSuggestionSelect(
                            track,
                          )
                        }
                      >
                        <span className="search-suggestion-cover">
                          {track.image_url ? (
                            <img
                              src={
                                track.image_url
                              }
                              alt=""
                            />
                          ) : (
                            <span>
                              ♪
                            </span>
                          )}
                        </span>

                        <span className="search-suggestion-info">
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
                        </span>

                        <span className="search-suggestion-genre">
                          {
                            track.genre
                          }
                        </span>
                      </button>
                    ),
                  )}
              </div>
            )}
          </div>

          <button
            type="submit"
            className="primary-button"
          >
            Search
          </button>
        </form>

        <div className="search-controls">
          <div
            className="search-tabs"
            aria-label="Search result type"
          >
            {(
              [
                ['all', 'All'],
                ['track', 'Tracks'],
                ['artist', 'Artists'],
                [
                  'release',
                  'Releases',
                ],
              ] as const
            ).map(
              ([
                value,
                label,
              ]) => (
                <button
                  key={value}
                  type="button"
                  className={`search-tab${
                    type === value
                      ? ' active'
                      : ''
                  }`}
                  onClick={() =>
                    handleTypeChange(
                      value,
                    )
                  }
                >
                  {label}
                </button>
              ),
            )}
          </div>

          <div className="search-filters">
            <label>
              <span>Sort</span>

              <select
                value={sort}
                onChange={(event) =>
                  handleSortChange(
                    event.target
                      .value as SearchSort,
                  )
                }
              >
                <option value="relevance">
                  Relevance
                </option>

                <option value="newest">
                  Newest
                </option>

                <option value="popular">
                  Popular
                </option>
              </select>
            </label>

            {(type === 'all' ||
              type === 'track') && (
              <label>
                <span>Genre</span>

                <input
                  type="text"
                  value={genre}
                  onChange={(event) =>
                    handleGenreChange(
                      event.target
                        .value,
                    )
                  }
                  placeholder="e.g. country"
                />
              </label>
            )}
          </div>
        </div>
      </section>

      {likeError && (
        <section className="content-panel">
          <p className="form-error">
            {likeError}
          </p>
        </section>
      )}

      {!query && (
        <section className="search-empty-state">
          <h3>
            Search the catalog
          </h3>

          <p>
            Enter a song, artist,
            release or genre above.
          </p>
        </section>
      )}

      {query && loading && (
        <section className="content-panel">
          <p>
            Searching for “{query}”...
          </p>
        </section>
      )}

      {query &&
        !loading &&
        error && (
          <section className="content-panel">
            <p className="form-error">
              {error}
            </p>
          </section>
        )}

      {query &&
        !loading &&
        !error &&
        result && (
          <>
            <section className="search-summary">
              <div>
                <p className="eyebrow">
                  RESULTS
                </p>

                <h3>
                  Results for “
                  {result.query}”
                </h3>
              </div>

              <span>
                {totalResults}{' '}
                {totalResults === 1
                  ? 'result'
                  : 'results'}
              </span>
            </section>

            {totalResults === 0 && (
              <section className="search-empty-state">
                <h3>
                  No results found
                </h3>

                <p>
                  Try another title,
                  artist, release or
                  genre.
                </p>
              </section>
            )}

            {(type === 'all' ||
              type === 'artist') &&
              result.artists.length >
                0 && (
                <section className="search-result-section">
                  <div className="section-heading">
                    <div>
                      <p className="eyebrow">
                        ARTISTS
                      </p>

                      <h3>
                        Artists
                      </h3>
                    </div>

                    <span className="text-button">
                      {
                        result
                          .pagination
                          .artist_total
                      }
                    </span>
                  </div>

                  <div className="search-artist-grid">
                    {result.artists.map(
                      (artist) => (
                        <Link
                          key={
                            artist.id
                          }
                          to={`/artists/${artist.id}`}
                          className="search-artist-card"
                        >
                          <span className="search-artist-avatar">
                            {artist.name
                              .charAt(0)
                              .toUpperCase()}
                          </span>

                          <strong>
                            {artist.name}
                          </strong>

                          <span>
                            Artist
                          </span>
                        </Link>
                      ),
                    )}
                  </div>
                </section>
              )}

            {(type === 'all' ||
              type ===
                'release') &&
              result.releases
                .length > 0 && (
                <section className="search-result-section">
                  <div className="section-heading">
                    <div>
                      <p className="eyebrow">
                        RELEASES
                      </p>

                      <h3>
                        Releases
                      </h3>
                    </div>

                    <span className="text-button">
                      {
                        result
                          .pagination
                          .release_total
                      }
                    </span>
                  </div>

                  <div className="search-release-grid">
                    {result.releases.map(
                      (release) => (
                        <article
                          key={
                            release.id
                          }
                          className="search-release-card"
                        >
                          <Link
                            to={`/releases/${release.id}`}
                            className="search-release-cover"
                          >
                            {release.cover_image_url ? (
                              <img
                                src={
                                  release.cover_image_url
                                }
                                alt={`${release.title} cover`}
                              />
                            ) : (
                              <span>
                                {
                                  release.release_type
                                }
                              </span>
                            )}
                          </Link>

                          <div className="search-release-info">
                            <Link
                              to={`/releases/${release.id}`}
                              className="search-release-title"
                            >
                              {
                                release.title
                              }
                            </Link>

                            <Link
                              to={`/artists/${release.artist_id}`}
                              className="artist-link"
                            >
                              {
                                release.artist_name
                              }
                            </Link>

                            <span>
                              {
                                release.release_type
                              }
                            </span>
                          </div>
                        </article>
                      ),
                    )}
                  </div>
                </section>
              )}

            {(type === 'all' ||
              type === 'track') &&
              result.tracks.length >
                0 && (
                <section className="search-result-section">
                  <div className="section-heading">
                    <div>
                      <p className="eyebrow">
                        TRACKS
                      </p>

                      <h3>
                        Tracks
                      </h3>
                    </div>

                    <span className="text-button">
                      {
                        result
                          .pagination
                          .track_total
                      }
                    </span>
                  </div>

                  <div className="search-track-list">
                    {result.tracks.map(
                      (track) => {
                        const isCurrentTrack =
                          currentTrack
                            ?.id ===
                          track.id

                        const isThisTrackPlaying =
                          isCurrentTrack &&
                          isPlaying

                        const trackIsLiked =
                          isLiked(
                            track.id,
                          )

                        const isLikePending =
                          pendingLikeTrackID ===
                          track.id

                        return (
                          <article
                            key={
                              track.id
                            }
                            className="library-track search-track-row"
                          >
                            <div className="library-track-cover">
                              {track.image_url && (
                                <img
                                  src={
                                    track.image_url
                                  }
                                  alt={`${track.song_title} cover`}
                                />
                              )}
                            </div>

                            <div className="library-track-info">
                              <strong>
                                {
                                  track.song_title
                                }
                              </strong>

                              <span>
                                {track.artist_id ? (
                                  <Link
                                    to={`/artists/${track.artist_id}`}
                                    className="artist-link"
                                  >
                                    {
                                      track.artist_name
                                    }
                                  </Link>
                                ) : (
                                  track.artist_name
                                )}
                              </span>
                            </div>

                            <span className="genre-pill">
                              {
                                track.genre
                              }
                            </span>

                            {track.audio_url ? (
                              <button
                                type="button"
                                className="track-play-button"
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
                            ) : (
                              <span className="search-track-no-audio">
                                —
                              </span>
                            )}

                            {isAuthenticated ? (
                              <AddToQueueButton
                                track={
                                  track
                                }
                              />
                            ) : (
                              <span />
                            )}

                            {isAuthenticated ? (
                              <AddToPlaylistButton
                                track={
                                  track
                                }
                              />
                            ) : (
                              <span />
                            )}

                            {isAuthenticated ? (
                              <button
                                type="button"
                                className={`music-like-button${
                                  trackIsLiked
                                    ? ' is-liked'
                                    : ''
                                }`}
                                aria-label={
                                  trackIsLiked
                                    ? `Unlike ${track.song_title}`
                                    : `Like ${track.song_title}`
                                }
                                aria-pressed={
                                  trackIsLiked
                                }
                                disabled={
                                  isLikePending
                                }
                                onClick={() =>
                                  void handleLike(
                                    track,
                                  )
                                }
                              >
                                <svg
                                  className="music-like-icon"
                                  viewBox="0 0 24 24"
                                  aria-hidden="true"
                                >
                                  <path
                                    d="M12 21s-7.2-4.35-9.5-8.5C.85 9.5 2.15 5.5 5.75 4.5c2.15-.6 4.15.25 5.25 1.85C12.1 4.75 14.1 3.9 16.25 4.5c3.6 1 4.9 5 3.25 8C17.2 16.65 12 21 12 21Z"
                                    fill={
                                      trackIsLiked
                                        ? 'currentColor'
                                        : 'none'
                                    }
                                    stroke="currentColor"
                                    strokeWidth="1.8"
                                    strokeLinecap="round"
                                    strokeLinejoin="round"
                                  />
                                </svg>
                              </button>
                            ) : (
                              <span />
                            )}
                          </article>
                        )
                      },
                    )}
                  </div>
                </section>
              )}

            {totalResults > 0 && (
              <section className="search-pagination">
                <button
                  type="button"
                  className="secondary-button"
                  disabled={
                    page <= 1
                  }
                  onClick={() =>
                    handlePageChange(
                      page - 1,
                    )
                  }
                >
                  Previous
                </button>

                <span>
                  Page {page}
                </span>

                <button
                  type="button"
                  className="secondary-button"
                  disabled={!hasMore}
                  onClick={() =>
                    handlePageChange(
                      page + 1,
                    )
                  }
                >
                  Next
                </button>
              </section>
            )}
          </>
        )}
    </>
  )
}

export default SearchPage