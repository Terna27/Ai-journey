import { useEffect, useRef } from "react";

import { usePlayer } from "../../context/PlayerContext";

type QueuePanelProps = {
  onClose: () => void;
};

function QueuePanel({ onClose }: QueuePanelProps) {
  const {
    queue,
    currentQueueIndex,
    isPlaying,
    playFromQueue,
    removeFromQueue,
    clearQueue,
    reorderQueue,
  } = usePlayer();

  const panelRef = useRef<HTMLDivElement | null>(null);

  /*
   * Close on Escape and on outside clicks.
   */
  useEffect(() => {
    function handleKeyDown(event: KeyboardEvent) {
      if (event.key === "Escape") {
        onClose();
      }
    }

    function handlePointerDown(event: PointerEvent) {
      const target = event.target as Node;

      if (panelRef.current && !panelRef.current.contains(target)) {
        // Clicks on the player bar itself
        // (including the queue toggle)
        // are handled by their own buttons.
        const playerBar = document.querySelector(".player");

        if (!playerBar?.contains(target)) {
          onClose();
        }
      }
    }

    document.addEventListener("keydown", handleKeyDown);

    document.addEventListener("pointerdown", handlePointerDown);

    return () => {
      document.removeEventListener("keydown", handleKeyDown);

      document.removeEventListener("pointerdown", handlePointerDown);
    };
  }, [onClose]);

  const hasCurrent = currentQueueIndex >= 0 && currentQueueIndex < queue.length;

  const upcomingCount = queue.length - (hasCurrent ? currentQueueIndex + 1 : 0);

  return (
    <div
      className="queue-panel"
      ref={panelRef}
      role="dialog"
      aria-label="Playback queue"
    >
      <div className="queue-panel-header">
        <div>
          <strong>Queue</strong>

          <small>
            {upcomingCount} {upcomingCount === 1 ? "item" : "items"} coming up
          </small>
        </div>

        <div className="queue-panel-actions">
          <button
            type="button"
            className="text-button"
            disabled={upcomingCount === 0}
            onClick={clearQueue}
          >
            Clear queue
          </button>

          <button type="button" className="text-button" onClick={onClose}>
            Close
          </button>
        </div>
      </div>

      {queue.length === 0 && (
        <div className="queue-empty">
          <p>Your queue is empty.</p>

          <p>Play a song or podcast episode, or add music to the queue.</p>
        </div>
      )}

      {queue.length > 0 && (
        <ul className="queue-list">
          {queue.map((item, index) => {
            const isCurrent = index === currentQueueIndex;

            return (
              <li
                key={`${item.key}-${index}`}
                className={isCurrent ? "queue-item is-current" : "queue-item"}
              >
                <button
                  type="button"
                  className="queue-item-main"
                  aria-label={`Play ${item.title}`}
                  onClick={() => playFromQueue(index)}
                >
                  <div className="queue-item-cover">
                    {item.artwork_url ? (
                      <img
                        src={item.artwork_url}
                        alt={`${item.title} artwork`}
                      />
                    ) : (
                      <span>♪</span>
                    )}
                  </div>

                  <div className="queue-item-info">
                    <strong>{item.title}</strong>

                    <span>{item.subtitle}</span>

                    {isCurrent && (
                      <em>{isPlaying ? "Now playing" : "Paused"}</em>
                    )}
                  </div>
                </button>

                <div className="queue-item-controls">
                  <button
                    type="button"
                    className="queue-icon-button"
                    aria-label={`Move ${item.title} up`}
                    disabled={index === 0}
                    onClick={() => reorderQueue(index, index - 1)}
                  >
                    ↑
                  </button>

                  <button
                    type="button"
                    className="queue-icon-button"
                    aria-label={`Move ${item.title} down`}
                    disabled={index === queue.length - 1}
                    onClick={() => reorderQueue(index, index + 1)}
                  >
                    ↓
                  </button>

                  <button
                    type="button"
                    className="queue-icon-button"
                    aria-label={`Remove ${item.title} from queue`}
                    onClick={() => removeFromQueue(index)}
                  >
                    ✕
                  </button>
                </div>
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}

export default QueuePanel;
