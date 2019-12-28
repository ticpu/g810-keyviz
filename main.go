package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"github.com/pkg/profile"

	g810 "github.com/ticpu/go-g810"
)

type CpuStats struct {
	Total int
	Idle int
}

func getCPUPercent(lk g810.LedKeyboard, lkLock sync.Mutex) {
	var cpuStatsOld [12]CpuStats
	var cpuStatsNew [12]CpuStats
	var keyMap [12]g810.KeyValue
	var cpuNo int
	var currentUsage uint8
	stat, err := os.Open("/proc/stat")

	if err != nil {
		log.Fatal(err)
		return
	}
	defer stat.Close()

	for {
		stat.Seek(0, 0)
		scanner := bufio.NewScanner(stat)
		cpuStatsMatch := regexp.MustCompile("^cpu([0-9]+)")
		for scanner.Scan() {
			cpuStats := strings.Split(scanner.Text(), " ")
			m := cpuStatsMatch.FindStringSubmatch(cpuStats[0])
			if m != nil {
				cpuNo, _ = strconv.Atoi(m[1])
				if cpuNo > len(cpuStatsOld) {
					log.Fatal("Too many CPUs")
					break
				}
				cpuIdle, _ := strconv.Atoi(cpuStats[4])
				cpuTotal := 0
				for _, statCol := range cpuStats[1:] {
					if statColInt, err := strconv.Atoi(statCol); err == nil {
						cpuTotal += statColInt
					}
					cpuStatsNew[cpuNo].Total = cpuTotal
					cpuStatsNew[cpuNo].Idle = cpuIdle
				}
			}
		}
		cpuNo++
		for n := 0; n < cpuNo; n++ {
			currentUsage = (uint8)((cpuStatsNew[n].Idle-cpuStatsOld[n].Idle)*215/(cpuStatsNew[n].Total-cpuStatsOld[n].Total))
			/*
			fmt.Printf("%d: %d/%d (%d/255)\n",
				n,
				cpuStatsNew[n].Idle-cpuStatsOld[n].Idle,
				cpuStatsNew[n].Total-cpuStatsOld[n].Total,
				currentUsage,
			)
			*/
			keyMap[n] = g810.KeyValue{
				g810.KeyF1+(g810.Key)(n),
				g810.KeyColor{0, 255-currentUsage, 0},
			}
			cpuStatsOld[n].Idle = cpuStatsNew[n].Idle
			cpuStatsOld[n].Total = cpuStatsNew[n].Total
		}
		//fmt.Println(keyMap[:cpuNo])
		lkLock.Lock()
		lk.SetKeys(keyMap[:cpuNo])
		lk.Commit()
		lkLock.Unlock()
		time.Sleep(200 * time.Millisecond)
	}
}

func main() {
	var keys []g810.KeyValue
	var n g810.Key

	defer profile.Start().Stop()

	for n = g810.KeyA; n < 1028+26; n++ {
		keys = append(keys, g810.KeyValue{n, g810.KeyColor{95,0,45}})
	}

	lk := g810.NewLedKeyboard()
	lkLock := sync.Mutex{}
	defer lk.Free()
	lk.Open()
	defer lk.Close()

	deviceInfo := lk.GetDeviceInfo()
	fmt.Printf(
		"Vendor: %s (0x%04X)\nProduct: %s (0x%04X)\nModel: %s\nS/N: %s\n",
		deviceInfo.Manufacturer, deviceInfo.VendorID,
		deviceInfo.Product, deviceInfo.ProductID,
		deviceInfo.KeyboardModel,
		deviceInfo.SerialNumber,
	)

	lk.SetAllKeys(g810.KeyColor{80, 0, 120})
	lk.SetGroupKeys(g810.GroupLogo, g810.KeyColor{0, 40, 40})
	lk.SetGroupKeys(g810.GroupFKeys, g810.KeyColor{0, 40, 0})
	lk.SetGroupKeys(g810.GroupArrows, g810.KeyColor{0, 120, 120})
	lk.SetKeys(keys)
	lk.Commit()

	getCPUPercent(lk, lkLock)
}
