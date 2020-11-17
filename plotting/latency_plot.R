install.packages("ggplot2", repos="http://cran.us.r-project.org")
install.packages("dplyr", repos="http://cran.us.r-project.org")

library(ggplot2)
library(dplyr)

args = commandArgs(trailingOnly=TRUE)

data = read.csv(args[1])
not_dropped <- filter(data, dropped == "false")
dropped <- filter(data, dropped == "true")
png(args[2])
plot <- ggplot() +
  geom_point(data=not_dropped, aes(x=time, y=latency, color=type)) +
  geom_vline(data=dropped, aes(xintercept=time)) + 
  ylim(0, 200) + 
  xlab("time (ms)") + 
  ylab("latency (ms)")
print(plot)
dev.off()
