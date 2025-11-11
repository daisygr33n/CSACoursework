import re
import pandas as pd 
import matplotlib.pyplot as plt
from striprtf.striprtf import rtf_to_text 

#change this to filename
filename = "/Users/daisygreen/GoProjects/GoProjects/python/bmLiveSdl.rtf"

#if it’s an RTF file extract plain text
if filename.endswith(".rtf"):
    with open(filename, "r", encoding="utf-8") as f:
        text = rtf_to_text(f.read())
else:
    with open(filename, "r", encoding="utf-8") as f:
        text = f.read()

pattern = re.compile(r"BenchmarkGol/\S+-(\d+)-\d+\s+\d+\s+(\d+)\s+ns/op")
data = [(int(t), int(ns)) for t, ns in re.findall(pattern, text)]

df = pd.DataFrame(data, columns=["Threads", "ns_per_op"]).sort_values("Threads")
df["ms_per_op"] = df["ns_per_op"] / 1e6

print(df)

plt.figure(figsize=(8, 5))
plt.bar(df["Threads"], df["ms_per_op"], color="skyblue", edgecolor="black")

plt.title("Go Game of Life Benchmark Results")
plt.xlabel("Number of Threads")
plt.ylabel("Time per Operation (ms/op)")
plt.grid(axis="y", linestyle="--", alpha=0.7)
plt.tight_layout()
plt.show()
