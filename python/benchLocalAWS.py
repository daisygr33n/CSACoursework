import numpy as np
import matplotlib.pyplot as plt
from mpl_toolkits.mplot3d import Axes3D

# Distributed on 4 local workers (ns/op)
distributed_local = [
    2368777917, 2329671583, 2337235625, 2324116834,
    2289062750, 2272764250, 2271636208, 2257759500,
    2227431375, 2217210125, 2236715625, 2243336750,
    2262296958, 2259609125, 2265114083, 2242661958
]

# Four AWS nodes (ns/op)
aws_nodes = [
    157271602333, 149482792833, 154613789917, 145561146875,
    152796148208, 175629316541, 165067595042, 168258257875,
    158766655834, 151071182875, 161959218958, 143678732709,
    144313530417, 160671640000, 151556590333, 147933932958
]

# X positions (benchmark runs)
x = np.arange(1, 17)

# Y positions (categories)
y_local = np.zeros_like(x)
y_aws = np.ones_like(x)

# Dimensions of each bar
dx = np.ones_like(x) * 0.3
dy = np.ones_like(x) * 0.3
dz_local = distributed_local
dz_aws = aws_nodes

# Create figure
fig = plt.figure(figsize=(12, 8))
ax = fig.add_subplot(111, projection='3d')

# Plot both datasets
ax.bar3d(x - 0.15, y_local, np.zeros_like(x), dx, dy, dz_local,
         color='royalblue', alpha=0.8, label='Distributed (4 local workers)')
ax.bar3d(x + 0.15, y_aws, np.zeros_like(x), dx, dy, dz_aws,
         color='tomato', alpha=0.8, label='Four AWS nodes')

# Labels
ax.set_title('Game of Life Benchmark Comparison (512x512x1000)', pad=20)
ax.set_xlabel('Benchmark Run')
ax.set_ylabel('Configuration')
ax.set_zlabel('Time (ns/op)')

# Custom Y-axis tick labels
ax.set_yticks([0, 1])
ax.set_yticklabels(['Local Distributed', 'AWS Nodes'])

# Rotate for a good view
ax.view_init(elev=25, azim=45)

# Legend and grid
ax.legend()
ax.grid(True)

plt.tight_layout()
plt.show()
