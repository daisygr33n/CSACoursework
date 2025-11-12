

import matplotlib.pyplot as plt

# --- Benchmark Data (ns/op) ---
optimised = [
    4789638584, 2864509541, 2260907542, 1967878833,
    2255199875, 2059048458, 1909485583, 1852847084,
    1930609542, 1862048625, 1836844625, 1717208542,
    1775665458, 1730401250, 1743236416, 1666349391
]

non_optimised = [
    4767987375, 2889268625, 2257261375, 1948098709,
    2261681416, 2094025041, 1913758708, 1818437375,
    1914270042, 1860116458, 1815460625, 1738961750,
    1799484500, 1819823958, 1754761333, 1676339791
]

x = list(range(1, 17))

# --- Plot ---
plt.figure(figsize=(10, 6))
plt.plot(x, optimised, marker='o', linewidth=2, label='Optimised Thread Split')
plt.plot(x, non_optimised, marker='s', linewidth=2, label='Non-Optimised Thread Split')

# --- Labels & Style ---
plt.title('Game of Life Benchmark: Optimised vs Non-Optimised Thread Split\n(512x512x1000)', fontsize=14, pad=15)
plt.xlabel('Benchmark Run', fontsize=12)
plt.ylabel('Time (ns/op)', fontsize=12)
plt.legend()
plt.grid(True, linestyle='--', alpha=0.6)

# ------------------------------
# OPTION 1: Zoom into the Y-axis range
# (Set min and max values to focus on area of difference)
plt.ylim(1.6e9, 2.4e9)
# ------------------------------
# OPTION 2: Use a logarithmic scale (comment out OPTION 1 to use)
# plt.yscale('log')
# ------------------------------
# OPTION 3: Normalize to relative difference (uncomment to compare % difference)
# base = [n / o for n, o in zip(non_optimised, optimised)]
# plt.figure()
# plt.plot(x, base, marker='o', label='Relative ratio (non-opt / opt)')
# plt.title('Relative Performance Ratio (Lower = Faster)')
# plt.xlabel('Benchmark Run')
# plt.ylabel('Ratio')
# plt.grid(True, linestyle='--', alpha=0.6)
# plt.legend()

plt.tight_layout()
plt.show()
