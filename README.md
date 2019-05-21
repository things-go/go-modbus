go modbus Supported formats
- modbus TCP Client
- modbus Serial(RTU,ASCII) Client
- modbus TCP Server

大量参考了![goburrow](https://github.com/goburrow/modbus)

Supported functions
-------------------
Bit access:
*   Read Discrete Inputs
*   Read Coils
*   Write Single Coil
*   Write Multiple Coils

16-bit access:
*   Read Input Registers
*   Read Holding Registers
*   Write Single Register
*   Write Multiple Registers
*   Read/Write Multiple Registers
*   Mask Write Register
*   Read FIFO Queue

References
----------
-   [Modbus Specifications and Implementation Guides](http://www.modbus.org/specs.php)
