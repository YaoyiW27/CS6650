import matplotlib.pyplot as plt

labels = ['Single Machine\n(Python)', 'Splitter', '3 Mappers\n(Parallel)', 'Reducer', 'MapReduce\nTotal']
times = [0.092, 0.247, 0.184, 0.193, 0.624]
colors = ['#4CAF50', '#2196F3', '#FF9800', '#9C27B0', '#F44336']

plt.figure(figsize=(10, 6))
bars = plt.bar(labels, times, color=colors, edgecolor='black', linewidth=0.5)

for bar, t in zip(bars, times):
    plt.text(bar.get_x() + bar.get_width()/2, bar.get_height() + 0.01,
             f'{t}s', ha='center', va='bottom', fontsize=12, fontweight='bold')

plt.title('MapReduce vs Single Machine Performance\nWord Count on Shakespeare\'s Hamlet (~159KB)', fontsize=14)
plt.ylabel('Time (seconds)', fontsize=12)
plt.ylim(0, 0.75)
plt.tight_layout()
plt.savefig('performance_chart.png', dpi=150)
plt.show()
print("Chart saved as performance_chart.png")