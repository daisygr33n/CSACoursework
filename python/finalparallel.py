import matplotlib.pyplot as plt
import numpy as np

# --- BenchmarkGol ns/op data ---
times = [
    4781662500, 2863612625, 2300809041, 1957630042,
    2265717709, 2066695042, 1917622000, 1820868417,
    1933725750, 1861568959, 1847754417, 1718936542,
    1784194125, 1736678750, 1762837750, 1684852167
]

# Convert to numpy array for easy math
data = np.array(times)

# --- Basic statistics ---
mean_val = np.mean(data)
variance_val = np.var(data)
std_dev_val = np.std(data)

# Print stats
print(f"Mean: {mean_val:.2f} ns/op")
print(f"Variance: {variance_val:.2f}")
print(f"Standard Deviation: {std_dev_val:.2f}")

# --- X-axis values ---
x = np.arange(1, len(data) + 1)

# --- Plot ---
plt.figure(figsize=(10, 6))
plt.plot(x, data, marker='o', linewidth=2, color='royalblue', label='BenchmarkGol (ns/op)')

# Mark mean line
plt.axhline(mean_val, color='red', linestyle='--', label=f'Mean = {mean_val:.2e} ns/op')

# Labels and styling
plt.title('BenchmarkGol Performance (512x512x1000)', fontsize=14, pad=15)
plt.xlabel('Benchmark Run', fontsize=12)
plt.ylabel('Time (ns/op)', fontsize=12)
plt.legend()
plt.grid(True, linestyle='--', alpha=0.6)
plt.tight_layout()

# Show plot
plt.show()
