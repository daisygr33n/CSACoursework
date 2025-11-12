import numpy as np
import matplotlib.pyplot as plt
from mpl_toolkits.mplot3d import Axes3D

# Benchmark results (ns/op)
distributed = [
    2368777917, 2329671583, 2337235625, 2324116834,
    2289062750, 2272764250, 2271636208, 2257759500,
    2227431375, 2217210125, 2236715625, 2243336750,
    2262296958, 2259609125, 2265114083, 2242661958
]

parallel_distributed = [
    5234702375, 5018076625, 5170741292, 5035724292,
    5107267375, 4951969250, 5068692625, 5014191292,
    5024694250, 5032575916, 5001293166, 5018189416,
    4955847250, 4944665625, 4960056417, 4976835208
]

# X positions (benchmark run numbers)
x = np.arange(1, 17)

# Y positions for the two series
y_dist = np.zeros_like(x)
y_par = np.ones_like(x)

# Bar width, depth, and height
dx = np.ones_like(x) * 0.3
dy = np.ones_like(x) * 0.3
dz_dist = distributed
dz_par = parallel_distributed

# Create 3D figure
fig = plt.figure(figsize=(12, 8))
ax = fig.add_subplot(111, projection='3d')

# Plot both sets of bars
ax.bar3d(x - 0.15, y_dist, np.zeros_like(x), dx, dy, dz_dist, color='pink', alpha=0.8, label='Distributed')
ax.bar3d(x + 0.15, y_par, np.zeros_like(x), dx, dy, dz_par, color='mediumaquamarine', alpha=0.8, label='Parallel Distributed')

# Customize axes
ax.set_title('3D Benchmark Comparison: Game of Life (512x512x1000)', pad=20)
ax.set_xlabel('Benchmark Run')
ax.set_ylabel('Configuration')
ax.set_zlabel('Time (ns/op)')

# Custom Y-axis labels
ax.set_yticks([0, 1])
ax.set_yticklabels(['Distributed', 'Parallel Distributed'])

# Rotate for best view
ax.view_init(elev=25, azim=45)

# Legend & grid
ax.legend()
ax.grid(True)

plt.tight_layout()
plt.show()
