import numpy as np
import matplotlib.pyplot as plt

data = np.loadtxt("spectrogram.csv", delimiter=",")

# transpose so frequency is on the y-axis, time on the x-axis
data = data.T

# log scale — raw magnitudes span a huge range, dB makes peaks visible
data_db = 20 * np.log10(data + 1e-6)

plt.figure(figsize=(12, 6))
plt.imshow(data_db, aspect="auto", origin="lower", cmap="magma")
plt.xlabel("Time (frame index)")
plt.ylabel("Frequency bin")
plt.colorbar(label="Magnitude (dB)")
plt.title("Spectrogram")
plt.savefig("spectrogram.png")
plt.show()