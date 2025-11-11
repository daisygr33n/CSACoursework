import matplotlib.pyplot as plt
import numpy as np
from mpl_toolkits.mplot3d import Axes3D

# ---------- Benchmark data (ns/op) ----------
times_1thread = [
    6262980333, 6214204916, 6234065167, 6249941500, 6245249125, 6222719209,
    6222133875, 6201415833, 6352543875, 6304073833, 6224937916, 6215774500,
    6535440708, 6229375459, 6669752500, 6701500333
]

times_8threads = [
    2844880458, 2808710250, 2877286208, 2814504500, 2909303375, 2865425625,
    2820798375, 2749335667, 2825670208, 2739804625, 2753699625, 2856898458,
    2784361167, 2698552459, 2746160375, 2697290584
]

# ---------- Convert to milliseconds ----------
ms_1thread = np.array(times_1thread) / 1e6
ms_8threads = np.array(times_8threads) / 1e6

# ---------- Setup 3D axes ----------
fig = plt.figure(figsize=(10, 6))
ax = fig.add_subplot(111, projection='3d')

# X positions (trial numbers)
xpos = np.arange(1, len(ms_1thread) + 1)
ypos_1 = np.zeros_like(xpos)          # y=0 for 1-thread bars
ypos_8 = np.ones_like(xpos) * 0.6     # y=0.6 for 8-thread bars (slightly offset)

# Bar dimensions
dx = 0.3
dy = 0.3

# Bar heights
dz_1 = ms_1thread
dz_8 = ms_8threads

# ---------- Draw bars ----------
ax.bar3d(xpos, ypos_1, np.zeros_like(dz_1), dx, dy, dz_1, color='royalblue', label='1 Thread')
ax.bar3d(xpos, ypos_8, np.zeros_like(dz_8), dx, dy, dz_8, color='darkorange', label='8 Threads')

# ---------- Styling ----------
ax.set_title('Game of Life Benchmark Comparison', pad=20)
ax.set_xlabel('Trial Number')
ax.set_ylabel('Threads')
ax.set_zlabel('Benchmark Time (ms)')
ax.set_yticks([0, 0.6])
ax.set_yticklabels(['1 Thread', '8 Threads'])

ax.view_init(elev=20, azim=-60)
ax.legend(loc='upper right')

plt.tight_layout()
plt.show()
