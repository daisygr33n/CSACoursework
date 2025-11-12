import matplotlib.pyplot as plt

# --- Benchmark Data (ns/op) ---
without_modulo = [
    4858726833, 2883478583, 2259372791, 1978190208,
    2297114417, 2067537250, 1924015833, 1824939084,
    1934961167, 1864426250, 1843481959, 1744441875,
    1833637541, 1728517916, 1754183084, 1683975208
]

with_modulo = [
    7148793500, 4094337250, 3097146333, 2610642208,
    2723430375, 2501034791, 2263693833, 2127362167,
    2306494042, 2191719375, 2126333375, 2069215958,
    2287394958, 2172508833, 2107478417, 2066368167
]

# X-axis (benchmark run numbers)
x = list(range(1, 17))

# --- Plot ---
plt.figure(figsize=(10, 6))
plt.plot(x, with_modulo, marker='o', linewidth=2, label='With Modulo')
plt.plot(x, without_modulo, marker='s', linewidth=2, label='Without Modulo')

# Labels and title
plt.title('Game of Life Benchmark Comparison (With vs Without Modulo)\n(512x512x1000)', fontsize=14, pad=15)
plt.xlabel('Benchmark Run', fontsize=12)
plt.ylabel('Time (ns/op)', fontsize=12)

# Grid, legend, and layout
plt.grid(True, linestyle='--', alpha=0.6)
plt.legend()
plt.tight_layout()

# Show plot
plt.show()
