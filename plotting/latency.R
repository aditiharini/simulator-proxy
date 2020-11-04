install.packages("ggplot2", repos="http://cran.us.r-project.org")

library(ggplot2)
args = commandArgs(trailingOnly=TRUE)

data = read.csv(args[1])
png(args[2])
plot <- ggplot() +
  geom_point(data=data, aes(x=time, y=latency, color=type)) +
  ylim(0, 200) + 
  xlab("time (ms)") + 
  ylab("latency (ms)")
print(plot)
dev.off()
  