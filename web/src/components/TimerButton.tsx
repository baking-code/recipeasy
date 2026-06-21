import { useState, useEffect, useRef } from "react";
import styles from "./TimerButton.module.css";

interface TimerButtonProps {
  minutes: number;
  label?: string;
}

type TimerState = "idle" | "running" | "paused" | "done";

export default function TimerButton({ minutes, label }: TimerButtonProps) {
  const [state, setState] = useState<TimerState>("idle");
  const [remaining, setRemaining] = useState(minutes * 60);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const totalSeconds = minutes * 60;

  useEffect(() => {
    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current);
    };
  }, []);

  const start = () => {
    if (state === "idle") setRemaining(totalSeconds);
    setState("running");
    intervalRef.current = setInterval(() => {
      setRemaining((r) => {
        if (r <= 1) {
          clearInterval(intervalRef.current!);
          setState("done");
          fireNotification(label ?? `${minutes} min timer`);
          return 0;
        }
        return r - 1;
      });
    }, 1000);
  };

  const pause = () => {
    if (intervalRef.current) clearInterval(intervalRef.current);
    setState("paused");
  };

  const reset = () => {
    if (intervalRef.current) clearInterval(intervalRef.current);
    setState("idle");
    setRemaining(totalSeconds);
  };

  const fmt = (secs: number) => {
    const m = Math.floor(secs / 60);
    const s = secs % 60;
    return `${m}:${s.toString().padStart(2, "0")}`;
  };

  const progress = state === "idle" ? 0 : 1 - remaining / totalSeconds;

  return (
    <div className={styles.wrap}>
      <div className={styles.ring}>
        <svg viewBox="0 0 36 36" className={styles.svg}>
          <circle className={styles.track} cx="18" cy="18" r="15.9" />
          <circle
            className={styles.fill}
            cx="18"
            cy="18"
            r="15.9"
            strokeDasharray={`${progress * 100} 100`}
            style={{ stroke: state === "done" ? "#22c55e" : "var(--color-primary)" }}
          />
        </svg>
        <span className={styles.time}>
          {state === "done" ? "✓" : fmt(remaining)}
        </span>
      </div>
      <div className={styles.controls}>
        <span className={styles.timerLabel}>{label ?? `${minutes} min`}</span>
        <div className={styles.btns}>
          {state === "idle" && (
            <button className={styles.btn} onClick={start}>Start</button>
          )}
          {state === "running" && (
            <button className={styles.btn} onClick={pause}>Pause</button>
          )}
          {state === "paused" && (
            <button className={styles.btn} onClick={start}>Resume</button>
          )}
          {(state === "paused" || state === "done") && (
            <button className={`${styles.btn} ${styles.ghost}`} onClick={reset}>Reset</button>
          )}
        </div>
      </div>
    </div>
  );
}

function fireNotification(title: string) {
  if (!("Notification" in window)) return;
  if (Notification.permission === "granted") {
    new Notification("Timer done!", { body: title, icon: "/icon-192.png" });
  } else if (Notification.permission !== "denied") {
    Notification.requestPermission().then((p) => {
      if (p === "granted") {
        new Notification("Timer done!", { body: title, icon: "/icon-192.png" });
      }
    });
  }
}
