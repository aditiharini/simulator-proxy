install.packages("ggplot2", repos="http://cran.us.r-project.org")
install.packages("dplyr", repos="http://cran.us.r-project.org")

library(ggplot2)
library(dplyr)

args = commandArgs(trailingOnly=TRUE)

data = read.csv(args[1])
not_dropped <- filter(data, dropped == "false")
separate_links <- filter(not_dropped, type == args[2])

plot <- ggplot() +
  geom_point(data=separate_links, aes(x=time, y=latency, color=type, shape=".")) +
  guides(color=FALSE) + 
  guides(shape=FALSE) + 
  xlab("time (ms)") + 
  ylab("latency (ms)")
ggsave(args[3], plot=plot)



