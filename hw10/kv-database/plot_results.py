"""
Generate graphs for KV Database Load Test Report.
Usage: python plot_results.py
Requires: pip install pandas matplotlib
Reads CSVs from results/ directory, outputs PNGs to graphs/ directory.
"""

import os
import glob
import pandas as pd
import matplotlib.pyplot as plt
import matplotlib.ticker as ticker
import numpy as np

# ============================================================
# Config
# ============================================================
RESULTS_DIR = "results"
GRAPHS_DIR = "graphs"
os.makedirs(GRAPHS_DIR, exist_ok=True)

DB_LABELS = {
    "leader-w5r1": "Leader W=5,R=1",
    "leader-w1r5": "Leader W=1,R=5",
    "leader-w3r3": "Leader W=3,R=3",
    "leaderless":  "Leaderless W=5,R=1",
}

WRITE_PCTS = [1, 10, 50, 90]
COLORS = {
    "leader-w5r1": "#2196F3",
    "leader-w1r5": "#FF9800",
    "leader-w3r3": "#4CAF50",
    "leaderless":  "#E91E63",
}

# ============================================================
# Load all data
# ============================================================
def load_all():
    frames = []
    for f in glob.glob(os.path.join(RESULTS_DIR, "*.csv")):
        df = pd.read_csv(f)
        frames.append(df)
    if not frames:
        print("No CSV files found in results/")
        return pd.DataFrame()
    return pd.concat(frames, ignore_index=True)

# ============================================================
# Graph 1 & 2: Latency distributions (reads and writes)
# One figure per write_pct, showing all 4 db configs
# Uses histogram with log-scale x-axis to show long tail
# ============================================================
def plot_latency_distributions(df):
    for wpct in WRITE_PCTS:
        subset = df[df["write_pct"] == wpct]
        if subset.empty:
            continue

        for req_type in ["read", "write"]:
            fig, axes = plt.subplots(2, 2, figsize=(14, 10))
            fig.suptitle(
                f"{req_type.capitalize()} Latency Distribution — "
                f"Write {wpct}% / Read {100-wpct}%",
                fontsize=16, fontweight="bold"
            )

            for idx, (db_type, label) in enumerate(DB_LABELS.items()):
                ax = axes[idx // 2][idx % 2]
                data = subset[(subset["db_type"] == db_type) & (subset["type"] == req_type)]

                if data.empty or len(data) < 2:
                    ax.text(0.5, 0.5, "No data", ha="center", va="center",
                            transform=ax.transAxes, fontsize=12)
                    ax.set_title(label)
                    continue

                latencies = data["latency_ms"].values
                latencies = latencies[latencies > 0]  # filter zeros for log

                if len(latencies) == 0:
                    ax.text(0.5, 0.5, "No data", ha="center", va="center",
                            transform=ax.transAxes, fontsize=12)
                    ax.set_title(label)
                    continue

                # Use log-spaced bins to show long tail
                min_lat = max(latencies.min(), 0.01)
                max_lat = latencies.max() * 1.1
                bins = np.logspace(np.log10(min_lat), np.log10(max_lat), 50)

                ax.hist(latencies, bins=bins, color=COLORS[db_type],
                        alpha=0.7, edgecolor="white", linewidth=0.5)
                ax.set_xscale("log")
                ax.set_title(f"{label}  (n={len(latencies)}, "
                             f"median={np.median(latencies):.1f}ms, "
                             f"p99={np.percentile(latencies, 99):.1f}ms)")
                ax.set_xlabel("Latency (ms)")
                ax.set_ylabel("Count")
                ax.xaxis.set_major_formatter(ticker.ScalarFormatter())
                ax.grid(axis="y", alpha=0.3)

            plt.tight_layout(rect=[0, 0, 1, 0.95])
            fname = f"{GRAPHS_DIR}/latency_{req_type}_w{wpct}.png"
            plt.savefig(fname, dpi=150, bbox_inches="tight")
            plt.close()
            print(f"  Saved {fname}")

# ============================================================
# Graph 3: Read-write time interval distribution
# For each key, measures time between consecutive write and read
# ============================================================
def plot_rw_intervals(df):
    for wpct in WRITE_PCTS:
        subset = df[df["write_pct"] == wpct].copy()
        if subset.empty:
            continue

        fig, axes = plt.subplots(2, 2, figsize=(14, 10))
        fig.suptitle(
            f"Read-Write Interval Distribution — "
            f"Write {wpct}% / Read {100-wpct}%",
            fontsize=16, fontweight="bold"
        )

        for idx, (db_type, label) in enumerate(DB_LABELS.items()):
            ax = axes[idx // 2][idx % 2]
            data = subset[subset["db_type"] == db_type].copy()

            if data.empty:
                ax.text(0.5, 0.5, "No data", ha="center", va="center",
                        transform=ax.transAxes, fontsize=12)
                ax.set_title(label)
                continue

            data["ts"] = pd.to_datetime(data["timestamp"])
            data = data.sort_values("ts")

            intervals = []
            last_write_time = {}

            for _, row in data.iterrows():
                key = row["key"]
                if row["type"] == "write":
                    last_write_time[key] = row["ts"]
                elif row["type"] == "read" and key in last_write_time:
                    delta_ms = (row["ts"] - last_write_time[key]).total_seconds() * 1000
                    if delta_ms >= 0:
                        intervals.append(delta_ms)

            if len(intervals) < 2:
                ax.text(0.5, 0.5, "Insufficient data", ha="center", va="center",
                        transform=ax.transAxes, fontsize=12)
                ax.set_title(label)
                continue

            intervals = np.array(intervals)
            intervals = intervals[intervals > 0]

            if len(intervals) == 0:
                ax.text(0.5, 0.5, "No intervals > 0", ha="center", va="center",
                        transform=ax.transAxes, fontsize=12)
                ax.set_title(label)
                continue

            min_iv = max(intervals.min(), 0.01)
            max_iv = intervals.max() * 1.1
            bins = np.logspace(np.log10(min_iv), np.log10(max_iv), 50)

            ax.hist(intervals, bins=bins, color=COLORS[db_type],
                    alpha=0.7, edgecolor="white", linewidth=0.5)
            ax.set_xscale("log")
            ax.set_title(f"{label}  (n={len(intervals)}, "
                         f"median={np.median(intervals):.1f}ms)")
            ax.set_xlabel("Interval between write and subsequent read (ms)")
            ax.set_ylabel("Count")
            ax.xaxis.set_major_formatter(ticker.ScalarFormatter())
            ax.grid(axis="y", alpha=0.3)

        plt.tight_layout(rect=[0, 0, 1, 0.95])
        fname = f"{GRAPHS_DIR}/rw_interval_w{wpct}.png"
        plt.savefig(fname, dpi=150, bbox_inches="tight")
        plt.close()
        print(f"  Saved {fname}")

# ============================================================
# Graph 4: Stale reads summary bar chart
# ============================================================
def plot_stale_reads(df):
    fig, ax = plt.subplots(figsize=(12, 6))

    x = np.arange(len(WRITE_PCTS))
    width = 0.2
    offsets = [-1.5, -0.5, 0.5, 1.5]

    for i, (db_type, label) in enumerate(DB_LABELS.items()):
        stale_pcts = []
        for wpct in WRITE_PCTS:
            data = df[(df["db_type"] == db_type) & (df["write_pct"] == wpct) & (df["type"] == "read")]
            if len(data) == 0:
                stale_pcts.append(0)
            else:
                stale_pcts.append(data["stale_read"].sum() / len(data) * 100)

        ax.bar(x + offsets[i] * width, stale_pcts, width,
               label=label, color=COLORS[db_type], alpha=0.8)

    ax.set_xlabel("Write Percentage", fontsize=12)
    ax.set_ylabel("Stale Read %", fontsize=12)
    ax.set_title("Stale Read Rate by Configuration and Write Ratio",
                 fontsize=14, fontweight="bold")
    ax.set_xticks(x)
    ax.set_xticklabels([f"W={w}%/R={100-w}%" for w in WRITE_PCTS])
    ax.legend()
    ax.grid(axis="y", alpha=0.3)

    fname = f"{GRAPHS_DIR}/stale_reads_summary.png"
    plt.savefig(fname, dpi=150, bbox_inches="tight")
    plt.close()
    print(f"  Saved {fname}")

# ============================================================
# Graph 5: Average latency comparison
# ============================================================
def plot_avg_latency_comparison(df):
    fig, (ax1, ax2) = plt.subplots(1, 2, figsize=(16, 6))
    fig.suptitle("Average Latency Comparison", fontsize=16, fontweight="bold")

    x = np.arange(len(WRITE_PCTS))
    width = 0.2
    offsets = [-1.5, -0.5, 0.5, 1.5]

    for req_type, ax, title in [("read", ax1, "Read Latency"), ("write", ax2, "Write Latency")]:
        for i, (db_type, label) in enumerate(DB_LABELS.items()):
            avgs = []
            for wpct in WRITE_PCTS:
                data = df[(df["db_type"] == db_type) & (df["write_pct"] == wpct) & (df["type"] == req_type)]
                avgs.append(data["latency_ms"].mean() if len(data) > 0 else 0)

            ax.bar(x + offsets[i] * width, avgs, width,
                   label=label, color=COLORS[db_type], alpha=0.8)

        ax.set_xlabel("Write Percentage", fontsize=12)
        ax.set_ylabel("Avg Latency (ms)", fontsize=12)
        ax.set_title(title, fontsize=13)
        ax.set_xticks(x)
        ax.set_xticklabels([f"W={w}%/R={100-w}%" for w in WRITE_PCTS])
        ax.legend(fontsize=9)
        ax.grid(axis="y", alpha=0.3)

    plt.tight_layout(rect=[0, 0, 1, 0.93])
    fname = f"{GRAPHS_DIR}/avg_latency_comparison.png"
    plt.savefig(fname, dpi=150, bbox_inches="tight")
    plt.close()
    print(f"  Saved {fname}")

# ============================================================
# Main
# ============================================================
if __name__ == "__main__":
    print("Loading data...")
    df = load_all()
    if df.empty:
        exit(1)

    print(f"Loaded {len(df)} records\n")

    print("Generating latency distributions...")
    plot_latency_distributions(df)

    print("\nGenerating read-write interval distributions...")
    plot_rw_intervals(df)

    print("\nGenerating stale reads summary...")
    plot_stale_reads(df)

    print("\nGenerating average latency comparison...")
    plot_avg_latency_comparison(df)

    print(f"\nAll graphs saved to {GRAPHS_DIR}/")
    print(f"Total files: {len(os.listdir(GRAPHS_DIR))}")